import { useState } from "react";
import { useParams } from "react-router-dom";
import { api } from "../api/client";

export default function Submit() {
  const { token } = useParams();
  const [guestName, setGuestName] = useState("");
  const [note, setNote] = useState("");
  const [images, setImages] = useState([]);
  const [audio, setAudio] = useState(null);
  const [step, setStep] = useState("form"); // form -> preview -> done
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  function handleReview(e) {
    e.preventDefault();
    if (!guestName.trim()) {
      setError("Please enter your name.");
      return;
    }
    setError("");
    setStep("preview"); // guest previews & confirms before it actually sends
  }

  async function handleConfirm() {
    setSubmitting(true);
    setError("");
    try {
      const formData = new FormData();
      formData.append("guest_name", guestName);
      formData.append("note", note);
      images.forEach((img) => formData.append("images", img));
      if (audio) formData.append("audio", audio);

      await api.submitEntry(token, formData);
      setStep("done");
    } catch (err) {
      setError(err.message);
    } finally {
      setSubmitting(false);
    }
  }

  if (step === "done") {
    return (
      <div className="submit-page">
        <div className="submit-card">
          <h1>Thanks, {guestName}!</h1>
          <p>Your autograph has been sent and is waiting for approval.</p>
        </div>
      </div>
    );
  }

  if (step === "preview") {
    return (
      <div className="submit-page">
        <div className="submit-card">
          <h1>Review before sending</h1>
          <div className="preview-block">
            <strong>{guestName}</strong>
            {note && <p>{note}</p>}
            {images.length > 0 && <p>{images.length} photo(s) attached</p>}
            {audio && <p>Audio message attached</p>}
          </div>
          {error && <p className="error">{error}</p>}
          <div className="entry-actions">
            <button onClick={() => setStep("form")} disabled={submitting}>Edit</button>
            <button onClick={handleConfirm} disabled={submitting}>
              {submitting ? "Sending…" : "Looks good, send it"}
            </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="submit-page">
      <form className="submit-card" onSubmit={handleReview}>
        <h1>Leave your autograph</h1>
        {error && <p className="error">{error}</p>}
        <label>
          Your name
          <input value={guestName} onChange={(e) => setGuestName(e.target.value)} required />
        </label>
        <label>
          Note
          <textarea value={note} onChange={(e) => setNote(e.target.value)} rows={4} />
        </label>
        <label>
          Photos
          <input type="file" accept="image/*" multiple onChange={(e) => setImages(Array.from(e.target.files))} />
        </label>
        <label>
          Audio message (optional)
          <input type="file" accept="audio/*" onChange={(e) => setAudio(e.target.files[0] || null)} />
        </label>
        <button type="submit">Preview</button>
      </form>
    </div>
  );
}
