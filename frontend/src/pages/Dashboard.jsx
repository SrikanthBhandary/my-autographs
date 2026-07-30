import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { api } from "../api/client";

const TYPES = ["school", "college", "company", "gym", "custom"];

export default function Dashboard() {
  const [categories, setCategories] = useState([]);
  const [name, setName] = useState("");
  const [type, setType] = useState("school");
  const [parentId, setParentId] = useState("");
  const [error, setError] = useState("");
  const [linkByCategory, setLinkByCategory] = useState({});

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
    try {
      const res = await api.createShareLink({ category_id: categoryId });
      setLinkByCategory((prev) => ({ ...prev, [categoryId]: res.url }));
    } catch (err) {
      setError(err.message);
    }
  }

  async function handleDelete(id) {
    if (!confirm("Delete this category and all its entries?")) return;
    try {
      await api.deleteCategory(id);
      refresh();
    } catch (err) {
      setError(err.message);
    }
  }

  const topLevel = categories.filter((c) => !c.parent_id);
  const childrenOf = (id) => categories.filter((c) => c.parent_id === id);

  return (
    <div className="page">
      <header className="page-header">
        <h1>My autographs</h1>
        <nav>
          <Link to="/review">Review submissions</Link>
          <Link to="/book">View book</Link>
        </nav>
      </header>

      {error && <p className="error">{error}</p>}

      <form className="inline-form" onSubmit={handleCreate}>
        <input placeholder="Category name (e.g. School A)" value={name} onChange={(e) => setName(e.target.value)} required />
        <select value={type} onChange={(e) => setType(e.target.value)}>
          {TYPES.map((t) => (
            <option key={t} value={t}>{t}</option>
          ))}
        </select>
        <select value={parentId} onChange={(e) => setParentId(e.target.value)}>
          <option value="">No parent (top-level group)</option>
          {topLevel.map((c) => (
            <option key={c.id} value={c.id}>{c.name}</option>
          ))}
        </select>
        <button type="submit">Add category</button>
      </form>

      <ul className="category-tree">
        {topLevel.map((cat) => (
          <li key={cat.id}>
            <div className="category-row">
              <strong>{cat.name}</strong> <span className="tag">{cat.type}</span>
              <button onClick={() => handleGenerateLink(cat.id)}>Get share link</button>
              <button className="danger" onClick={() => handleDelete(cat.id)}>Delete</button>
            </div>
            {linkByCategory[cat.id] && (
              <p className="share-url">
                {linkByCategory[cat.id]}{" "}
                <button onClick={() => navigator.clipboard.writeText(linkByCategory[cat.id])}>Copy</button>
              </p>
            )}
            <ul>
              {childrenOf(cat.id).map((child) => (
                <li key={child.id}>
                  <div className="category-row">
                    {child.name}
                    <button onClick={() => handleGenerateLink(child.id)}>Get share link</button>
                    <button className="danger" onClick={() => handleDelete(child.id)}>Delete</button>
                  </div>
                  {linkByCategory[child.id] && (
                    <p className="share-url">
                      {linkByCategory[child.id]}{" "}
                      <button onClick={() => navigator.clipboard.writeText(linkByCategory[child.id])}>Copy</button>
                    </p>
                  )}
                </li>
              ))}
            </ul>
          </li>
        ))}
      </ul>
    </div>
  );
}
