import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { api } from "../api/client";

export default function Book() {
  const [entries, setEntries] = useState([]);
  const [index, setIndex] = useState(0);
  const [error, setError] = useState("");

  useEffect(() => {
    api.listEntries("approved").then(setEntries).catch((err) => setError(err.message));
  }, []);

  const entry = entries[index];

  return (
    <div className="page book-page">
      <header className="page-header">
        <h1>My autograph book</h1>
        <nav>
          <Link to="/dashboard">Back to dashboard</Link>
        </nav>
      </header>

      {error && <p className="error">{error}</p>}
      {entries.length === 0 && !error && <p className="empty">No approved autographs yet.</p>}

      {entry && (
        <div className="book">
          <button className="book-nav" onClick={() => setIndex((i) => Math.max(0, i - 1))} disabled={index === 0}>
            ‹
          </button>

          <div className="book-leaf">
            {entry.image_urls?.[0] && <img src={entry.image_urls[0]} alt="" />}
            <h2>{entry.guest_name}</h2>
            {entry.note && <p>{entry.note}</p>}
            {entry.audio_url && <audio controls src={entry.audio_url} />}
            <span className="page-number">{index + 1} / {entries.length}</span>
          </div>

          <button className="book-nav" onClick={() => setIndex((i) => Math.min(entries.length - 1, i + 1))} disabled={index === entries.length - 1}>
            ›
          </button>
        </div>
      )}
    </div>
  );
}
