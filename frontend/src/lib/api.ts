const ACCESS_KEY = "thingspan_access";
const REFRESH_KEY = "thingspan_refresh";

export function isLoggedIn(): boolean {
  return !!localStorage.getItem(ACCESS_KEY);
}

export function clearAuth(): void {
  localStorage.removeItem(ACCESS_KEY);
  localStorage.removeItem(REFRESH_KEY);
}

export function logout(): void {
  clearAuth();
  window.location.href = "/login";
}

export async function login(password: string): Promise<void> {
  const res = await fetch("/api/auth/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ password }),
  });
  const body = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(body.detail || "登录失败");
  }
  localStorage.setItem(ACCESS_KEY, body.access_token);
  localStorage.setItem(REFRESH_KEY, body.refresh_token);
}

let refreshing: Promise<boolean> | null = null;

async function tryRefresh(): Promise<boolean> {
  const refreshToken = localStorage.getItem(REFRESH_KEY);
  if (!refreshToken) return false;
  const res = await fetch("/api/auth/refresh", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ refresh_token: refreshToken }),
  });
  if (!res.ok) return false;
  const body = await res.json();
  localStorage.setItem(ACCESS_KEY, body.access_token);
  localStorage.setItem(REFRESH_KEY, body.refresh_token);
  return true;
}

export async function api<T>(path: string, options: RequestInit = {}): Promise<T> {
  const headers = new Headers(options.headers);
  headers.set("Content-Type", "application/json");
  const token = localStorage.getItem(ACCESS_KEY);
  if (token) headers.set("Authorization", `Bearer ${token}`);

  const res = await fetch(path, { ...options, headers });

  if (res.status === 401 && !path.startsWith("/api/auth/")) {
    refreshing = refreshing ?? tryRefresh().finally(() => {
      refreshing = null;
    });
    const ok = await refreshing;
    if (ok) return api<T>(path, options);
    clearAuth();
    window.location.href = "/login";
    throw new Error("登录已过期，请重新登录");
  }

  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.detail || `请求失败（${res.status}）`);
  }
  return res.json() as Promise<T>;
}
