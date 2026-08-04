// Package queue wraps RabbitMQ for two purposes:
//
//  1. A durable work queue ("pdf_generation_jobs") that the API publishes
//     to and one or more worker processes consume from — standard
//     competing-consumers pattern, so you can run multiple workers for
//     throughput without any code changes.
//
//  2. A fanout exchange ("pdf_notifications") that the worker publishes
//     completion events to. Every API instance gets its own copy of every
//     notification (via an anonymous, auto-deleted queue bound to the
//     exchange) and forwards it to the browser over WebSocket if it happens
//     to be holding that user's connection. This is what makes it safe to
//     run more than one API instance — a plain queue would only deliver
//     each notification to ONE instance, which might not be the one with
//     the relevant user connected.
package queue

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	PDFJobQueueName         = "pdf_generation_jobs"
	PDFNotificationExchange = "pdf_notifications"
)

type Queue struct {
	conn      *amqp.Connection
	publishCh *amqp.Channel
	consumeCh *amqp.Channel
}

func Connect(url string) (*Queue, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("connecting to rabbitmq: %w", err)
	}

	publishCh, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("opening publish channel: %w", err)
	}
	consumeCh, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("opening consume channel: %w", err)
	}

	for _, ch := range []*amqp.Channel{publishCh, consumeCh} {
		if _, err := ch.QueueDeclare(PDFJobQueueName, true, false, false, false, nil); err != nil {
			conn.Close()
			return nil, fmt.Errorf("declaring job queue: %w", err)
		}
		if err := ch.ExchangeDeclare(PDFNotificationExchange, "fanout", true, false, false, false, nil); err != nil {
			conn.Close()
			return nil, fmt.Errorf("declaring notification exchange: %w", err)
		}
	}

	return &Queue{conn: conn, publishCh: publishCh, consumeCh: consumeCh}, nil
}

func (q *Queue) Close() error {
	if q.publishCh != nil {
		q.publishCh.Close()
	}
	if q.consumeCh != nil {
		q.consumeCh.Close()
	}
	if q.conn != nil {
		return q.conn.Close()
	}
	return nil
}

// --- Job queue: API publishes, Worker(s) consume ---

type PDFJobMessage struct {
	JobID string `json:"job_id"`
}

func (q *Queue) PublishPDFJob(ctx context.Context, jobID string) error {
	body, err := json.Marshal(PDFJobMessage{JobID: jobID})
	if err != nil {
		return err
	}
	return q.publishCh.PublishWithContext(ctx, "", PDFJobQueueName, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent, // survive a RabbitMQ restart
		Body:         body,
	})
}

// ConsumePDFJobs is used by the worker. Qos(1) means a worker only ever has
// one unacknowledged job in flight at a time — with multiple worker
// processes running, RabbitMQ spreads jobs across them instead of one
// worker grabbing a big batch while others sit idle.
func (q *Queue) ConsumePDFJobs(ctx context.Context) (<-chan amqp.Delivery, error) {
	if err := q.consumeCh.Qos(1, 0, false); err != nil {
		return nil, err
	}
	return q.consumeCh.ConsumeWithContext(ctx, PDFJobQueueName, "", false, false, false, false, nil)
}

// --- Notification fanout: Worker publishes, every API instance consumes ---

type JobNotification struct {
	UserID  string  `json:"user_id"`
	JobID   string  `json:"job_id"`
	Status  string  `json:"status"`
	FileURL *string `json:"file_url,omitempty"`
	Error   *string `json:"error,omitempty"`
}

func (q *Queue) PublishNotification(ctx context.Context, n JobNotification) error {
	body, err := json.Marshal(n)
	if err != nil {
		return err
	}
	return q.publishCh.PublishWithContext(ctx, PDFNotificationExchange, "", false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	})
}

// ConsumeNotifications gives this process its own anonymous, auto-delete
// queue bound to the fanout exchange, so it (and every other running API
// instance) gets a copy of every notification independently.
func (q *Queue) ConsumeNotifications(ctx context.Context) (<-chan amqp.Delivery, error) {
	dq, err := q.consumeCh.QueueDeclare("", false, true, true, false, nil)
	if err != nil {
		return nil, err
	}
	if err := q.consumeCh.QueueBind(dq.Name, "", PDFNotificationExchange, false, nil); err != nil {
		return nil, err
	}
	return q.consumeCh.ConsumeWithContext(ctx, dq.Name, "", true, false, false, false, nil)
}
