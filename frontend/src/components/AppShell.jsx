import { Feather, LayoutGrid, ClipboardCheck, BookOpen, Cloud, LogOut } from "lucide-react";
import { NavLink } from "react-router-dom";
import { useAuth } from "../context/AuthContext";
import { useTheme, THEMES } from "../context/ThemeContext";

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
  const { theme, setTheme } = useTheme();
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
          <NavLink to="/wordcloud" className={({ isActive }) => (isActive ? "active" : "")}>
            <Cloud size={16} /> Word cloud
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

        <div className="sidebar-footer">
          <div className="theme-picker">
            <span className="eyebrow">Theme</span>
            <div className="theme-swatches">
              {THEMES.map((t) => (
                <button
                  key={t.id}
                  type="button"
                  className={`theme-swatch${theme === t.id ? " selected" : ""}`}
                  style={{ background: `linear-gradient(135deg, ${t.swatch[0]} 50%, ${t.swatch[1]} 50%)` }}
                  onClick={() => setTheme(t.id)}
                  title={t.name}
                  aria-label={`${t.name} theme`}
                />
              ))}
            </div>
          </div>

          <button className="ghost" onClick={logout}>
            <LogOut size={16} /> {user?.name || "Log out"}
          </button>
        </div>
      </aside>

      <main className="app-main">{children}</main>
    </div>
  );
}
