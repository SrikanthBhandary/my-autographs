import { useEffect, useMemo, useState } from "react";
import { ChevronLeft, ChevronRight, BookOpen } from "lucide-react";
import AppShell from "../components/AppShell";
import { api } from "../api/client";

export default function Book() {
  const [categories, setCategories] = useState([]);
  const [allEntries, setAllEntries] = useState([]);
  const [selectedId, setSelectedId] = useState(null);
  const [index, setIndex] = useState(0);
  const [error, setError] = useState("");

  useEffect(() => {
    Promise.all([api.listCategories(), api.listEntries("approved")])
      .then(([cats, entries]) => {
        setCategories(cats);
        setAllEntries(entries);
      })
      .catch((err) => setError(err.message));
  }, []);

  // Selecting a top-level category (e.g. "School") should also include
  // entries from its sub-categories (e.g. "School A", "School B") — so we
  // walk the tree to collect every descendant id under whatever is selected.
  const scopedCategoryIds = useMemo(() => {
    if (!selectedId) return null; // null = no filter, show everything
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

  const entries = scopedCategoryIds
    ? allEntries.filter((e) => scopedCategoryIds.has(e.category_id))
    : allEntries;

  const selectedCategory = categories.find((c) => c.id === selectedId);
  const entry = entries[index];

  function selectCategory(id) {
    setSelectedId(id);
    setIndex(0);
  }

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

        {entry && (
          <div className="book">
            <button className="book-nav icon-btn" onClick={() => setIndex((i) => Math.max(0, i - 1))} disabled={index === 0}>
              <ChevronLeft size={28} />
            </button>

            <div className="book-leaf">
              {entry.image_urls?.[0] && <img src={entry.image_urls[0]} alt="" />}
              <h2>{entry.guest_name}</h2>
              {entry.note && <p>{entry.note}</p>}
              {entry.audio_url && <audio controls src={entry.audio_url} />}
              <span className="page-number">{index + 1} / {entries.length}</span>
            </div>

            <button className="book-nav icon-btn" onClick={() => setIndex((i) => Math.min(entries.length - 1, i + 1))} disabled={index === entries.length - 1}>
              <ChevronRight size={28} />
            </button>
          </div>
        )}
      </div>
    </AppShell>
  );
}
