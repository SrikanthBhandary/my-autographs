const API_URL = import.meta.env.VITE_API_URL || "http://localhost:8080/api";

function authHeaders() {
  const token = localStorage.getItem("token");
  return token ? { Authorization: `Bearer ${token}` } : {};
}

async function handle(res) {
  if (!res.ok) {
    let message = `Request failed (${res.status})`;
    try {
      const body = await res.json();
      if (body.error) message = body.error;
    } catch {
      // no JSON body
    }
    throw new Error(message);
  }
  if (res.status === 204) return null;
  return res.json();
}

export const api = {
  // --- auth ---
  signup: (data) =>
    fetch(`${API_URL}/auth/signup`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(data),
    }).then(handle),

  login: (data) =>
    fetch(`${API_URL}/auth/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(data),
    }).then(handle),

  // --- categories ---
  listCategories: () =>
    fetch(`${API_URL}/categories`, { headers: authHeaders() }).then(handle),

  createCategory: (data) =>
    fetch(`${API_URL}/categories`, {
      method: "POST",
      headers: { "Content-Type": "application/json", ...authHeaders() },
      body: JSON.stringify(data),
    }).then(handle),

  deleteCategory: (id) =>
    fetch(`${API_URL}/categories/${id}`, {
      method: "DELETE",
      headers: authHeaders(),
    }).then(handle),

  // --- share links ---
  createShareLink: (data) =>
    fetch(`${API_URL}/sharelinks`, {
      method: "POST",
      headers: { "Content-Type": "application/json", ...authHeaders() },
      body: JSON.stringify(data),
    }).then(handle),

  // --- entries ---
  listEntries: (status, categoryId) => {
    const params = new URLSearchParams();
    if (status) params.set("status", status);
    if (categoryId) params.set("category_id", categoryId);
    return fetch(`${API_URL}/entries?${params}`, { headers: authHeaders() }).then(handle);
  },

  approveEntry: (id) =>
    fetch(`${API_URL}/entries/${id}/approve`, { method: "PATCH", headers: authHeaders() }).then(handle),

  rejectEntry: (id) =>
    fetch(`${API_URL}/entries/${id}/reject`, { method: "PATCH", headers: authHeaders() }).then(handle),

  // --- public guest submission (no auth) ---
  submitEntry: (token, formData) =>
    fetch(`${API_URL}/submit/${token}`, {
      method: "POST",
      body: formData, // multipart/form-data — don't set Content-Type manually
    }).then(handle),
};
