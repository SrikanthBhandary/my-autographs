import { useEffect, useState } from "react";
import { Check, X, ClipboardCheck } from "lucide-react";
import AppShell from "../components/AppShell";
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
    <AppShell>
      <header className="main-header">
        <span className="eyebrow">Moderation</span>
        <h1>Review submissions</h1>
        <p>Approve entries to add them to your book, or reject anything you'd rather not keep.</p>
      </header>

      {error && <p className="error">{error}</p>}

      {entries.length === 0 && !error && (
        <div className="empty-panel">
          <ClipboardCheck size={36} />
          <p>Nothing waiting for review right now.</p>
        </div>
      )}

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
              <button onClick={() => approve(entry.id)}><Check size={16} /> Approve</button>
              <button className="danger" onClick={() => reject(entry.id)}><X size={16} /> Reject</button>
            </div>
          </li>
        ))}
      </ul>
    </AppShell>
  );
}
