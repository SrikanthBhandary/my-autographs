# Autograph App — Frontend (React + Vite)

## Setup

```bash
cp .env.example .env    # points at your backend, default http://localhost:8080/api
npm install
npm run dev              # http://localhost:5173
```

## Pages

| Route | Auth | Purpose |
|---|---|---|
| `/signup`, `/login` | none | owner account creation / login |
| `/dashboard` | owner | manage categories (School, College, Company, Gym...), generate share links |
| `/review` | owner | approve/reject pending guest submissions |
| `/book` | owner | flip through approved autographs |
| `/submit/:token` | none (public) | guest opens a share link, fills the form, previews, and sends |

## Notes

- The JWT is stored in `localStorage` for simplicity. For production, consider
  moving to an httpOnly cookie issued by the backend to reduce XSS exposure.
- `src/api/client.js` is the single place all backend calls go through —
  when you build the React Native app later, this file (with `fetch` swapped
  for React Native's built-in `fetch`, which has the same API) is the piece
  you can port almost as-is.
- Styling is plain CSS in `src/index.css` — intentionally simple so it's easy
  to restyle once you're happy with the functionality.
