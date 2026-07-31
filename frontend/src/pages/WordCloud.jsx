import { useEffect, useMemo, useRef, useState } from "react";
import { Cloud } from "lucide-react";
import AppShell from "../components/AppShell";
import { api } from "../api/client";

const STOPWORDS = new Set([
  "the","a","an","and","or","but","if","then","so","because","as","of","at",
  "by","for","with","about","against","between","into","through","during",
  "before","after","above","below","to","from","up","down","in","out","on",
  "off","over","under","again","further","once","here","there","when","where",
  "why","how","all","any","both","each","few","more","most","other","some",
  "such","no","nor","not","only","own","same","than","too","very","can",
  "will","just","should","now","is","are","was","were","be","been","being",
  "have","has","had","having","do","does","did","doing","would","could",
  "you","your","yours","yourself","yourselves","i","im","ive","me","my",
  "mine","myself","we","our","ours","ourselves","he","him","his","himself",
  "she","her","hers","herself","it","its","itself","they","them","their",
  "theirs","themselves","this","that","these","those","what","which","who",
  "whom","also","really","get","got","one","us","gonna","gunna","lot","much",
]);

function extractWords(text) {
  return text
    .toLowerCase()
    .replace(/['’]/g, "")
    .split(/[^a-z0-9]+/i)
    .filter((w) => w.length >= 3 && !STOPWORDS.has(w) && !/^\d+$/.test(w));
}

function buildFrequencies(entries, limit = 50) {
  const freq = new Map();
  entries.forEach((e) => {
    if (!e.note) return;
    extractWords(e.note).forEach((w) => freq.set(w, (freq.get(w) || 0) + 1));
  });
  return [...freq.entries()]
    .sort((a, b) => b[1] - a[1])
    .slice(0, limit)
    .map(([text, count]) => ({ text, count }));
}

const PALETTE = ["#2f5d50", "#b8935f", "#1c1e2a", "#6b8f83", "#a97c3f"];
const CANVAS_W = 820;
const CANVAS_H = 460;
const FONT_FAMILY = "Fraunces, Georgia, serif";

// Classic spiral-placement word cloud layout: place the biggest word first
// at the center, then for each subsequent word walk an outward spiral until
// its bounding box no longer overlaps anything already placed.
async function layoutWords(words) {
  if (document.fonts?.ready) await document.fonts.ready;

  const canvas = document.createElement("canvas");
  const ctx = canvas.getContext("2d");
  const maxCount = words[0]?.count || 1;
  const minCount = words[words.length - 1]?.count || 1;

  function fontSizeFor(count) {
    const t = maxCount === minCount ? 1 : (count - minCount) / (maxCount - minCount);
    return Math.round(14 + Math.sqrt(t) * 40); // 14px .. 54px, sqrt scale so it's not too extreme
  }

  const placed = [];
  const center = { x: CANVAS_W / 2, y: CANVAS_H / 2 };

  words.forEach((word, i) => {
    const size = fontSizeFor(word.count);
    ctx.font = `600 ${size}px ${FONT_FAMILY}`;
    const w = ctx.measureText(word.text).width + 6;
    const h = size * 1.15;

    let angle = Math.random() * Math.PI * 2;
    let radius = 0;
    let x = center.x - w / 2;
    let y = center.y - h / 2;
    let attempts = 0;
    let ok = false;

    while (!ok && attempts < 4000) {
      const box = { x, y, w, h };
      const inBounds = box.x >= 0 && box.y >= 0 && box.x + box.w <= CANVAS_W && box.y + box.h <= CANVAS_H;
      const collides = placed.some(
        (p) => !(box.x + box.w < p.x || p.x + p.w < box.x || box.y + box.h < p.y || p.y + p.h < box.y)
      );
      if (inBounds && !collides) {
        ok = true;
      } else {
        angle += 0.28;
        radius += 2.0;
        x = center.x + radius * Math.cos(angle) - w / 2;
        y = center.y + radius * Math.sin(angle) * 0.7 - h / 2; // flatten vertically to fill a wide card
        attempts++;
      }
    }

    if (ok) {
      placed.push({
        x, y, w, h,
        text: word.text,
        count: word.count,
        size,
        color: PALETTE[i % PALETTE.length],
        rotate: Math.random() < 0.15 ? -90 : 0,
      });
    }
    // words that never find a free spot within the attempt budget are just
    // skipped — with a 60-word cap and a 760x420 canvas this is rare.
  });

  return placed;
}

export default function WordCloud() {
  const [categories, setCategories] = useState([]);
  const [allEntries, setAllEntries] = useState([]);
  const [selectedId, setSelectedId] = useState(null);
  const [placedWords, setPlacedWords] = useState(null);
  const [error, setError] = useState("");
  const layoutToken = useRef(0);

  useEffect(() => {
    Promise.all([api.listCategories(), api.listEntries("approved")])
      .then(([cats, entries]) => {
        setCategories(cats);
        setAllEntries(entries);
      })
      .catch((err) => setError(err.message));
  }, []);

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

  const entries = useMemo(
    () => (scopedCategoryIds ? allEntries.filter((e) => scopedCategoryIds.has(e.category_id)) : allEntries),
    [scopedCategoryIds, allEntries]
  );
  const frequencies = useMemo(() => buildFrequencies(entries), [entries]);
  const selectedCategory = categories.find((c) => c.id === selectedId);

  useEffect(() => {
    const token = ++layoutToken.current;
    if (frequencies.length === 0) {
      setPlacedWords([]);
      return;
    }
    setPlacedWords(null); // show loading state while we compute
    layoutWords(frequencies).then((placed) => {
      if (layoutToken.current === token) setPlacedWords(placed);
    });
  }, [frequencies]);

  return (
    <AppShell categories={categories} selectedId={selectedId} onSelectCategory={setSelectedId}>
      <header className="main-header">
        <span className="eyebrow">What people wrote</span>
        <h1>{selectedCategory ? selectedCategory.name : "Word cloud"}</h1>
        <p>The most common words across everyone's notes{selectedCategory ? ` in "${selectedCategory.name}"` : ""}.</p>
      </header>

      {error && <p className="error">{error}</p>}

      {frequencies.length === 0 && !error && (
        <div className="empty-panel">
          <Cloud size={36} />
          <p>Not enough notes yet to build a word cloud — approve a few submissions with notes first.</p>
        </div>
      )}

      {frequencies.length > 0 && (
        <section className="panel">
          <div className="wordcloud-frame">
            {placedWords === null ? (
              <p className="empty" style={{ textAlign: "center" }}>Arranging words…</p>
            ) : (
              <svg viewBox={`0 0 ${CANVAS_W} ${CANVAS_H}`} className="wordcloud-svg" role="img" aria-label="Word cloud of notes">
                {placedWords.map((w) => (
                  <text
                    key={w.text}
                    x={w.x + w.w / 2}
                    y={w.y + w.h / 2}
                    fontSize={w.size}
                    fill={w.color}
                    fontFamily={FONT_FAMILY}
                    fontWeight={600}
                    textAnchor="middle"
                    dominantBaseline="central"
                    transform={w.rotate ? `rotate(${w.rotate} ${w.x + w.w / 2} ${w.y + w.h / 2})` : undefined}
                  >
                    {w.text}
                    <title>{w.text} — {w.count}×</title>
                  </text>
                ))}
              </svg>
            )}
          </div>
        </section>
      )}

      {frequencies.length > 0 && (
        <section className="panel">
          <h2 className="panel-title">Top words</h2>
          <ol className="word-rank-list">
            {frequencies.slice(0, 10).map((w) => (
              <li key={w.text}>
                <span>{w.text}</span>
                <span className="word-rank-count">{w.count}×</span>
              </li>
            ))}
          </ol>
        </section>
      )}
    </AppShell>
  );
}
