import { useEffect, useState } from "react";
import { Plus, Link2, Copy, Trash2, FolderPlus } from "lucide-react";
import AppShell from "../components/AppShell";
import { api } from "../api/client";

const TYPES = [
  { value: "school", label: "School" },
  { value: "college", label: "College" },
  { value: "company", label: "Company" },
  { value: "gym", label: "Gym" },
  { value: "custom", label: "Custom" },
];

export default function Dashboard() {
  const [categories, setCategories] = useState([]);
  const [selectedId, setSelectedId] = useState(null);
  const [name, setName] = useState("");
  const [type, setType] = useState("school");
  const [parentId, setParentId] = useState("");
  const [error, setError] = useState("");
  const [shareUrl, setShareUrl] = useState(null);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    refresh();
  }, []);

  async function refresh() {
    try {
      const data = await api.listCategories();
      setCategories(data);
    } catch (err) {
      setError(err.message);
    }
  }

  async function handleCreate(e) {
    e.preventDefault();
    setError("");
    try {
      await api.createCategory({ name, type, parent_id: parentId || null });
      setName("");
      setParentId("");
      refresh();
    } catch (err) {
      setError(err.message);
    }
  }

  async function handleGenerateLink(categoryId) {
    setCopied(false);
    try {
      const res = await api.createShareLink({ category_id: categoryId });
      setShareUrl(res.url);
    } catch (err) {
      setError(err.message);
    }
  }

  async function handleDelete(id) {
    if (!confirm("Delete this category and all its entries?")) return;
    try {
      await api.deleteCategory(id);
      if (selectedId === id) setSelectedId(null);
      refresh();
    } catch (err) {
      setError(err.message);
    }
  }

  function copyLink() {
    navigator.clipboard.writeText(shareUrl);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  }

  const selected = categories.find((c) => c.id === selectedId);
  const topLevel = categories.filter((c) => !c.parent_id);

  return (
    <AppShell categories={categories} selectedId={selectedId} onSelectCategory={(id) => { setSelectedId(id); setShareUrl(null); }}>
      <header className="main-header">
        <span className="eyebrow">My autographs</span>
        <h1>{selected ? selected.name : "All categories"}</h1>
        <p>Organize your keepsakes and generate links for people to sign.</p>
      </header>

      {error && <p className="error">{error}</p>}

      <section className="panel">
        <h2 className="panel-title"><FolderPlus size={18} /> Add a category</h2>
        <form onSubmit={handleCreate}>
          <div className="field-grid">
            <label>
              Name
              <input placeholder="e.g. School A, Acme Corp" value={name} onChange={(e) => setName(e.target.value)} required />
            </label>
            <label>
              Type
              <select value={type} onChange={(e) => setType(e.target.value)}>
                {TYPES.map((t) => (
                  <option key={t.value} value={t.value}>{t.label}</option>
                ))}
              </select>
            </label>
            <label>
              Parent group (optional)
              <select value={parentId} onChange={(e) => setParentId(e.target.value)}>
                <option value="">None — top-level group</option>
                {topLevel.map((c) => (
                  <option key={c.id} value={c.id}>{c.name}</option>
                ))}
              </select>
            </label>
          </div>
          <div className="panel-actions">
            <button type="submit"><Plus size={16} /> Add category</button>
          </div>
        </form>
      </section>

      {selected && (
        <section className="panel">
          <h2 className="panel-title"><Link2 size={18} /> Share link for "{selected.name}"</h2>
          <p style={{ color: "var(--ink-soft)", marginTop: 0 }}>
            Anyone with this link can leave an autograph in this category — no account needed on their end.
          </p>
          <div className="panel-actions" style={{ justifyContent: "flex-start" }}>
            <button onClick={() => handleGenerateLink(selected.id)}><Link2 size={16} /> Generate link</button>
            <button className="danger" onClick={() => handleDelete(selected.id)}><Trash2 size={16} /> Delete category</button>
          </div>
          {shareUrl && (
            <div className="share-url-box">
              <span>{shareUrl}</span>
              <button className="ghost" onClick={copyLink}><Copy size={14} /> {copied ? "Copied!" : "Copy"}</button>
            </div>
          )}
        </section>
      )}

      {!selected && categories.length === 0 && (
        <div className="empty-panel">
          <FolderPlus size={36} />
          <p>No categories yet — add your first one above to get started.</p>
        </div>
      )}
    </AppShell>
  );
}
