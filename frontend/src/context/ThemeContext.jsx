import { createContext, useContext, useEffect, useState } from "react";

export const THEMES = [
  { id: "keepsake", name: "Keepsake", swatch: ["#2f5d50", "#b8935f"] },
  { id: "midnight", name: "Midnight", swatch: ["#17181d", "#5fa08c"] },
  { id: "blush", name: "Blush", swatch: ["#a8506b", "#c9a15a"] },
  { id: "ocean", name: "Ocean", swatch: ["#1f6f78", "#c08a4e"] },
  { id: "sunset", name: "Sunset", swatch: ["#c1562c", "#d4a039"] },
];

const ThemeContext = createContext(null);

export function ThemeProvider({ children }) {
  const [theme, setTheme] = useState(() => {
    const saved = localStorage.getItem("theme");
    return THEMES.some((t) => t.id === saved) ? saved : "keepsake";
  });

  useEffect(() => {
    document.documentElement.setAttribute("data-theme", theme);
    localStorage.setItem("theme", theme);
  }, [theme]);

  return <ThemeContext.Provider value={{ theme, setTheme }}>{children}</ThemeContext.Provider>;
}

export function useTheme() {
  const ctx = useContext(ThemeContext);
  if (!ctx) throw new Error("useTheme must be used within ThemeProvider");
  return ctx;
}
