import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { api } from "../api/client";

export default function Review() {
  const [entries, setEntries] = useState([]);
  const [error, setError] = useState("");

  useEffect(() => {
    refresh();
  }, []);

  async function refresh() {
    try {
      setEntries(await api.listEntries("pending"));
    } catch (err) {
      setError(err.message);
    }
  }

  async function approve(id) {
    await api.approveEntry(id);
    refresh();
  }

  async function reject(id) {
    await api.rejectEntry(id);
    refresh();
  }

  return (
    <div className="page">
      <header className="page-header">
        <h1>Review submissions</h1>
        <nav>
          <Link to="/dashboard">Back to dashboard</Link>
        </nav>
      </header>

      {error && <p className="error">{error}</p>}
      {entries.length === 0 && <p className="empty">Nothing waiting for review right now.</p>}

      <ul className="entry-list">
        {entries.map((entry) => (
          <li key={entry.id} className="entry-card">
            <div className="entry-meta">
              <strong>{entry.guest_name}</strong>
              <span>{new Date(entry.created_at).toLocaleString()}</span>
            </div>
            {entry.note && <p>{entry.note}</p>}
            {entry.image_urls?.length > 0 && (
              <div className="thumb-row">
                {entry.image_urls.map((url) => (
                  <img key={url} src={url} alt="" />
                ))}
              </div>
            )}
            {entry.audio_url && <audio controls src={entry.audio_url} />}
            <div className="entry-actions">
              <button onClick={() => approve(entry.id)}>Approve</button>
              <button className="danger" onClick={() => reject(entry.id)}>Reject</button>
            </div>
          </li>
        ))}
      </ul>
    </div>
  );
}
