import { useEffect, useMemo, useRef, useState } from "react";
import HTMLFlipBook from "react-pageflip";
import { ChevronLeft, ChevronRight, BookOpen, Feather } from "lucide-react";
import AppShell from "../components/AppShell";
import Page from "../components/Page";
import { api } from "../api/client";

export default function Book() {
  const [categories, setCategories] = useState([]);
  const [allEntries, setAllEntries] = useState([]);
  const [selectedId, setSelectedId] = useState(null);
  const [pageNum, setPageNum] = useState(0);
  const [error, setError] = useState("");
  const bookRef = useRef(null);

  useEffect(() => {
    Promise.all([api.listCategories(), api.listEntries("approved")])
      .then(([cats, entries]) => {
        setCategories(cats);
        setAllEntries(entries);
      })
      .catch((err) => setError(err.message));
  }, []);

  // Selecting a top-level category also pulls in its sub-categories' entries.
  const scopedCategoryIds = useMemo(() => {
    if (!selectedId) return null;
    const ids = new Set([selectedId]);
    let added = true;
    while (added) {
      added = false;
      for (const c of categories) {
        if (c.parent_id && ids.has(c.parent_id) && !ids.has(c.id)) {
          ids.add(c.id);
          added = true;
        }
      }
    }
    return ids;
  }, [selectedId, categories]);

  const entries = scopedCategoryIds ? allEntries.filter((e) => scopedCategoryIds.has(e.category_id)) : allEntries;
  const selectedCategory = categories.find((c) => c.id === selectedId);

  function selectCategory(id) {
    setSelectedId(id);
    setPageNum(0);
    bookRef.current?.pageFlip()?.turnToPage(0);
  }

  const totalLeaves = entries.length + 2; // + front cover + back cover
  const atStart = pageNum === 0;
  const atEnd = pageNum >= totalLeaves - 1;

  return (
    <AppShell categories={categories} selectedId={selectedId} onSelectCategory={selectCategory}>
      <div className="book-page">
        <header className="main-header">
          <span className="eyebrow">Your keepsake</span>
          <h1>{selectedCategory ? selectedCategory.name : "My autograph book"}</h1>
          <p>{entries.length} approved {entries.length === 1 ? "entry" : "entries"}</p>
        </header>

        {error && <p className="error">{error}</p>}

        {entries.length === 0 && !error && (
          <div className="empty-panel">
            <BookOpen size={36} />
            <p>
              {selectedCategory
                ? `No approved autographs in "${selectedCategory.name}" yet.`
                : "No approved autographs yet — approve some from the Review page."}
            </p>
          </div>
        )}

        {entries.length > 0 && (
          <div className="ibooks-shelf">
            <button
              className="ibooks-nav"
              onClick={() => bookRef.current?.pageFlip()?.flipPrev()}
              disabled={atStart}
              aria-label="Previous page"
            >
              <ChevronLeft size={22} />
            </button>

            <HTMLFlipBook
              key={selectedId || "all"}
              ref={bookRef}
              width={300}
              height={460}
              size="fixed"
              minWidth={260}
              maxWidth={340}
              minHeight={400}
              maxHeight={520}
              maxShadowOpacity={0.4}
              showCover={true}
              mobileScrollSupport={false}
              className="ibooks-flip"
              onFlip={(e) => setPageNum(e.data)}
            >
              <Page className="cover">
                <div className="cover-mark"><Feather size={26} /></div>
                <h2>{selectedCategory ? selectedCategory.name : "My Autograph Book"}</h2>
                <span className="cover-sub">{entries.length} signatures collected</span>
              </Page>

              {entries.map((entry) => (
                <Page key={entry.id}>
                  <div className="leaf-content">
                    {entry.image_urls?.[0] && (
                      <div className="leaf-photo">
                        <img src={entry.image_urls[0]} alt="" />
                      </div>
                    )}
                    <h3 className="leaf-name">{entry.guest_name}</h3>
                    {entry.note && (
                      <div className="leaf-note-scroll">
                        <p className="leaf-note">{entry.note}</p>
                      </div>
                    )}
                    {entry.audio_url && <audio className="leaf-audio" controls src={entry.audio_url} />}
                  </div>
                </Page>
              ))}

              <Page className="cover back-cover">
                <div className="cover-mark"><Feather size={22} /></div>
                <span className="cover-sub">The end</span>
              </Page>
            </HTMLFlipBook>

            <button
              className="ibooks-nav"
              onClick={() => bookRef.current?.pageFlip()?.flipNext()}
              disabled={atEnd}
              aria-label="Next page"
            >
              <ChevronRight size={22} />
            </button>
          </div>
        )}

        {entries.length > 0 && (
          <div className="ibooks-progress">
            <div className="ibooks-progress-track">
              <div
                className="ibooks-progress-fill"
                style={{ width: `${(pageNum / Math.max(totalLeaves - 1, 1)) * 100}%` }}
              />
            </div>
            <span>{Math.min(pageNum + 1, totalLeaves)} of {totalLeaves}</span>
          </div>
        )}
      </div>
    </AppShell>
  );
}
