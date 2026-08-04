// Command worker is a separate process from the API server. It consumes
// PDF export jobs from RabbitMQ, builds the actual PDF (fetching entry
// images over plain HTTP since they're already public URLs), uploads the
// result to S3, and publishes a completion notification back over the
// fanout exchange so any API instance holding the owner's WebSocket
// connection can push it down to their browser.
//
// Run one or more of these alongside the API — RabbitMQ spreads jobs across
// however many are running (see queue.ConsumePDFJobs's Qos(1) setting).
package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/lib/pq"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/yourorg/autograph-backend/internal/config"
	"github.com/yourorg/autograph-backend/internal/db"
	"github.com/yourorg/autograph-backend/internal/queue"
	"github.com/yourorg/autograph-backend/internal/storage"
)

type worker struct {
	db      *sql.DB
	storage *storage.Storage
	queue   *queue.Queue
}

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	database, err := db.Connect(cfg.DB)
	if err != nil {
		log.Fatalf("db error: %v", err)
	}
	defer database.Close()

	store, err := storage.New(context.Background(), cfg.S3, os.Getenv("S3_PUBLIC_BASE_URL"))
	if err != nil {
		log.Fatalf("storage error: %v", err)
	}

	mq, err := queue.Connect(cfg.RabbitMQURL)
	if err != nil {
		log.Fatalf("rabbitmq error: %v", err)
	}
	defer mq.Close()

	w := &worker{db: database, storage: store, queue: mq}

	deliveries, err := mq.ConsumePDFJobs(context.Background())
	if err != nil {
		log.Fatalf("failed to start consuming jobs: %v", err)
	}

	log.Println("worker started, waiting for PDF export jobs...")
	for d := range deliveries {
		w.handleDelivery(d)
	}
}

func (w *worker) handleDelivery(d amqp.Delivery) {
	var msg queue.PDFJobMessage
	if err := json.Unmarshal(d.Body, &msg); err != nil {
		log.Printf("bad job message, dropping: %v", err)
		d.Ack(false)
		return
	}

	log.Printf("processing pdf job %s", msg.JobID)
	if err := w.processJob(msg.JobID); err != nil {
		log.Printf("job %s failed: %v", msg.JobID, err)
	}
	// Ack regardless of success/failure — the job's status/error is recorded
	// in the database and surfaced to the user; requeueing a job that keeps
	// failing (e.g. a permanently missing image) would just loop forever.
	d.Ack(false)
}

type jobEntry struct {
	guestName string
	note      string
	imageURL  *string
}

func (w *worker) processJob(jobID string) error {
	ctx := context.Background()

	var userID string
	var categoryID *string
	if err := w.db.QueryRowContext(ctx,
		`SELECT user_id, category_id FROM pdf_jobs WHERE id = $1`, jobID,
	).Scan(&userID, &categoryID); err != nil {
		return fmt.Errorf("loading job: %w", err)
	}

	if _, err := w.db.ExecContext(ctx, `UPDATE pdf_jobs SET status = 'processing' WHERE id = $1`, jobID); err != nil {
		return fmt.Errorf("marking job processing: %w", err)
	}

	title, entries, err := w.loadBookContent(ctx, userID, categoryID)
	if err != nil {
		w.failJob(ctx, jobID, userID, err)
		return err
	}

	pdfBytes, err := buildPDF(title, entries)
	if err != nil {
		w.failJob(ctx, jobID, userID, err)
		return err
	}

	key := fmt.Sprintf("exports/%s/%s.pdf", userID, uuid.NewString())
	fileURL, err := w.storage.UploadBytes(ctx, key, pdfBytes, "application/pdf")
	if err != nil {
		w.failJob(ctx, jobID, userID, err)
		return err
	}

	if _, err := w.db.ExecContext(ctx,
		`UPDATE pdf_jobs SET status = 'completed', file_url = $1, completed_at = now() WHERE id = $2`,
		fileURL, jobID,
	); err != nil {
		return fmt.Errorf("marking job completed: %w", err)
	}

	_ = w.queue.PublishNotification(ctx, queue.JobNotification{
		UserID:  userID,
		JobID:   jobID,
		Status:  "completed",
		FileURL: &fileURL,
	})
	log.Printf("job %s completed: %s", jobID, fileURL)
	return nil
}

func (w *worker) failJob(ctx context.Context, jobID, userID string, jobErr error) {
	errMsg := jobErr.Error()
	_, _ = w.db.ExecContext(ctx,
		`UPDATE pdf_jobs SET status = 'failed', error = $1, completed_at = now() WHERE id = $2`,
		errMsg, jobID,
	)
	_ = w.queue.PublishNotification(ctx, queue.JobNotification{
		UserID: userID,
		JobID:  jobID,
		Status: "failed",
		Error:  &errMsg,
	})
}

// loadBookContent fetches the title and approved entries to include in the
// export. A nil categoryID means "the whole book" (every category for this
// user); otherwise it includes that category and all of its sub-categories,
// matching the same scoping the frontend's Book/WordCloud views use.
func (w *worker) loadBookContent(ctx context.Context, userID string, categoryID *string) (string, []jobEntry, error) {
	title := "My Autograph Book"

	var categoryFilter []string
	if categoryID != nil {
		var name string
		if err := w.db.QueryRowContext(ctx, `SELECT name FROM categories WHERE id = $1 AND user_id = $2`, *categoryID, userID).Scan(&name); err != nil {
			return "", nil, fmt.Errorf("loading category: %w", err)
		}
		title = name

		ids, err := w.descendantCategoryIDs(ctx, userID, *categoryID)
		if err != nil {
			return "", nil, err
		}
		categoryFilter = ids
	}

	query := `
		SELECT e.guest_name, e.note, e.image_urls
		FROM entries e
		JOIN categories c ON c.id = e.category_id
		WHERE c.user_id = $1 AND e.status = 'approved'
	`
	args := []interface{}{userID}
	if categoryFilter != nil {
		query += ` AND e.category_id = ANY($2)`
		args = append(args, categoryFilter)
	}
	query += ` ORDER BY e.created_at ASC`

	rows, err := w.db.QueryContext(ctx, query, args...)
	if err != nil {
		return "", nil, fmt.Errorf("loading entries: %w", err)
	}
	defer rows.Close()

	var entries []jobEntry
	for rows.Next() {
		var e jobEntry
		var imageURLs []string
		if err := rows.Scan(&e.guestName, &e.note, pq.Array(&imageURLs)); err != nil {
			return "", nil, fmt.Errorf("reading entry: %w", err)
		}
		if len(imageURLs) > 0 {
			e.imageURL = &imageURLs[0]
		}
		entries = append(entries, e)
	}

	return title, entries, nil
}

// descendantCategoryIDs walks the category tree to include sub-categories
// (e.g. selecting "School" also pulls in "School A", "School B").
func (w *worker) descendantCategoryIDs(ctx context.Context, userID, rootID string) ([]string, error) {
	rows, err := w.db.QueryContext(ctx, `SELECT id, parent_id FROM categories WHERE user_id = $1`, userID)
	if err != nil {
		return nil, fmt.Errorf("loading categories: %w", err)
	}
	defer rows.Close()

	type cat struct {
		id       string
		parentID *string
	}
	var all []cat
	for rows.Next() {
		var c cat
		if err := rows.Scan(&c.id, &c.parentID); err != nil {
			return nil, err
		}
		all = append(all, c)
	}

	ids := map[string]bool{rootID: true}
	added := true
	for added {
		added = false
		for _, c := range all {
			if c.parentID != nil && ids[*c.parentID] && !ids[c.id] {
				ids[c.id] = true
				added = true
			}
		}
	}

	result := make([]string, 0, len(ids))
	for id := range ids {
		result = append(result, id)
	}
	return result, nil
}

// buildPDF lays out a title page followed by one page per entry.
func buildPDF(title string, entries []jobEntry) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetTitle(title, true)

	pdf.AddPage()
	pdf.SetFont("Times", "B", 28)
	pdf.SetY(120)
	pdf.CellFormat(0, 15, title, "", 1, "C", false, 0, "")
	pdf.SetFont("Times", "I", 12)
	pdf.CellFormat(0, 10, fmt.Sprintf("%d signatures collected", len(entries)), "", 1, "C", false, 0, "")

	for i, e := range entries {
		pdf.AddPage()
		y := 20.0

		if e.imageURL != nil {
			if imgName, imgType, ok := fetchImageForPDF(pdf, *e.imageURL, i); ok {
				pdf.ImageOptions(imgName, 65, y, 80, 0, false, fpdf.ImageOptions{ImageType: imgType}, 0, "")
				y += 90
			}
		}

		pdf.SetY(y)
		pdf.SetFont("Times", "B", 18)
		pdf.CellFormat(0, 10, e.guestName, "", 1, "C", false, 0, "")

		if e.note != "" {
			pdf.SetFont("Times", "", 13)
			pdf.SetY(pdf.GetY() + 4)
			pdf.SetX(30)
			pdf.MultiCell(150, 7, e.note, "", "C", false)
		}
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("rendering pdf: %w", err)
	}
	return buf.Bytes(), nil
}

// fetchImageForPDF downloads an already-public entry image over HTTP and
// registers it with fpdf under a unique in-memory name. Returns ok=false
// (and simply omits the image) if the fetch or decode fails, rather than
// failing the whole export over one bad image.
func fetchImageForPDF(pdf *fpdf.Fpdf, url string, index int) (name string, imgType string, ok bool) {
	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		log.Printf("skipping image (fetch error): %v", err)
		return "", "", false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("skipping image (status %d): %s", resp.StatusCode, url)
		return "", "", false
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("skipping image (read error): %v", err)
		return "", "", false
	}

	imgType = detectImageType(resp.Header.Get("Content-Type"), url)
	name = fmt.Sprintf("entry-image-%d", index)
	pdf.RegisterImageOptionsReader(name, fpdf.ImageOptions{ImageType: imgType}, bytes.NewReader(data))
	if pdf.Err() {
		log.Printf("skipping image (register error): %v", pdf.Error())
		pdf.ClearError()
		return "", "", false
	}
	return name, imgType, true
}

func detectImageType(contentType, url string) string {
	switch {
	case contains(contentType, "png"), hasSuffix(url, ".png"):
		return "PNG"
	case contains(contentType, "gif"), hasSuffix(url, ".gif"):
		return "GIF"
	default:
		return "JPG"
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
