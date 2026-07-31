import { useRef, useState } from "react";
import { useParams } from "react-router-dom";
import { Feather, Image, Mic, PenLine } from "lucide-react";
import SignaturePad from "../components/SignaturePad";
import { api } from "../api/client";

export default function Submit() {
  const { token } = useParams();
  const [guestName, setGuestName] = useState("");
  const [note, setNote] = useState("");
  const [images, setImages] = useState([]);
  const [audio, setAudio] = useState(null);
  const [signaturePreview, setSignaturePreview] = useState(null); // data URL, for display + reload on Edit
  const [signatureBlob, setSignatureBlob] = useState(null); // actual PNG blob to upload
  const [step, setStep] = useState("form"); // form -> preview -> done
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const signatureRef = useRef(null);

  // NOTE: this must be async — <SignaturePad> only exists in the "form"
  // step's JSX tree, so it unmounts the instant we call setStep("preview").
  // We have to read the canvas *before* that happens, or signatureRef.current
  // will be null by the time handleConfirm runs.
  async function handleReview(e) {
    e.preventDefault();
    if (!guestName.trim()) {
      setError("Please enter your name.");
      return;
    }
    setError("");

    if (signatureRef.current && !signatureRef.current.isEmpty()) {
      setSignaturePreview(signatureRef.current.toDataURL());
      const blob = await signatureRef.current.toBlob();
      setSignatureBlob(blob);
    } else {
      setSignaturePreview(null);
      setSignatureBlob(null);
    }

    setStep("preview");
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
      if (signatureBlob) formData.append("images", signatureBlob, "signature.png");

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
          <div className="brand"><span className="brand-mark"><Feather size={16} /></span> Keepsake</div>
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
            {signaturePreview && (
              <img src={signaturePreview} alt="Your signature" className="signature-preview-img" />
            )}
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
        <div className="brand"><span className="brand-mark"><Feather size={16} /></span> Keepsake</div>
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
          <span style={{ display: "flex", alignItems: "center", gap: "0.4em" }}><PenLine size={15} /> Sign or draw (optional)</span>
          {/* initialImage restores a previous drawing if the guest hit "Edit" and came back */}
          <SignaturePad ref={signatureRef} initialImage={signaturePreview} />
        </label>
        <label>
          <span style={{ display: "flex", alignItems: "center", gap: "0.4em" }}><Image size={15} /> Photos</span>
          <input type="file" accept="image/*" multiple onChange={(e) => setImages(Array.from(e.target.files))} />
        </label>
        <label>
          <span style={{ display: "flex", alignItems: "center", gap: "0.4em" }}><Mic size={15} /> Audio message (optional)</span>
          <input type="file" accept="audio/*" onChange={(e) => setAudio(e.target.files[0] || null)} />
        </label>
        <button type="submit">Preview</button>
      </form>
    </div>
  );
}
