import { Feather, LayoutGrid, ClipboardCheck, BookOpen, LogOut } from "lucide-react";
import { NavLink } from "react-router-dom";
import { useAuth } from "../context/AuthContext";

const STAMP_LETTER = { school: "S", college: "C", company: "Co", gym: "G", custom: "•" };

function CategoryNode({ category, children, selectedId, onSelect }) {
  return (
    <li>
      <div
        className={`category-nav-row${selectedId === category.id ? " selected" : ""}`}
        onClick={() => onSelect(category.id)}
      >
        <span className="stamp">{STAMP_LETTER[category.type] || "•"}</span>
        {category.name}
      </div>
      {children}
    </li>
  );
}

export default function AppShell({ children, categories = [], selectedId, onSelectCategory }) {
  const { user, logout } = useAuth();
  const topLevel = categories.filter((c) => !c.parent_id);
  const childrenOf = (id) => categories.filter((c) => c.parent_id === id);

  return (
    <div className="app-shell">
      <aside className="app-sidebar">
        <div className="brand">
          <span className="brand-mark"><Feather size={16} /></span>
          Keepsake
        </div>

        <nav className="sidebar-nav">
          <NavLink to="/dashboard" className={({ isActive }) => (isActive ? "active" : "")}>
            <LayoutGrid size={16} /> Dashboard
          </NavLink>
          <NavLink to="/review" className={({ isActive }) => (isActive ? "active" : "")}>
            <ClipboardCheck size={16} /> Review submissions
          </NavLink>
          <NavLink to="/book" className={({ isActive }) => (isActive ? "active" : "")}>
            <BookOpen size={16} /> View book
          </NavLink>
        </nav>

        {onSelectCategory && (
          <div className="category-nav-section">
            <span className="eyebrow">My autographs</span>
            <ul className="category-nav-tree">
              <li>
                <div
                  className={`category-nav-row${selectedId == null ? " selected" : ""}`}
                  onClick={() => onSelectCategory(null)}
                >
                  <span className="stamp">*</span>
                  All
                </div>
              </li>
              {topLevel.map((cat) => (
                <CategoryNode key={cat.id} category={cat} selectedId={selectedId} onSelect={onSelectCategory}>
                  {childrenOf(cat.id).length > 0 && (
                    <ul>
                      {childrenOf(cat.id).map((child) => (
                        <CategoryNode key={child.id} category={child} selectedId={selectedId} onSelect={onSelectCategory} />
                      ))}
                    </ul>
                  )}
                </CategoryNode>
              ))}
            </ul>
          </div>
        )}

        <button className="ghost" onClick={logout} style={{ marginTop: "auto" }}>
          <LogOut size={16} /> {user?.name || "Log out"}
        </button>
      </aside>

      <main className="app-main">{children}</main>
    </div>
  );
}
