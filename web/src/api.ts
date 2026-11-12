import type { HistoryEntry, Item, Page, Recommendation, SimilarItem, User } from "./types";

const API_URL = import.meta.env.VITE_API_URL ?? "http://localhost:8080";
const TOKEN_KEY = "token";

export class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message);
  }
}

export const tokenStore = {
  get: () => localStorage.getItem(TOKEN_KEY),
  set: (t: string) => localStorage.setItem(TOKEN_KEY, t),
  clear: () => localStorage.removeItem(TOKEN_KEY),
};

// Called when the API says the token is invalid, so the app can log out.
let onUnauthorized = () => {};
export function setUnauthorizedHandler(fn: () => void) {
  onUnauthorized = fn;
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = {};
  const token = tokenStore.get();
  if (token) headers.Authorization = `Bearer ${token}`;
  if (body !== undefined) headers["Content-Type"] = "application/json";

  let res: Response;
  try {
    res = await fetch(API_URL + path, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
    });
  } catch {
    throw new ApiError(0, "Cannot reach the server. Please try again.");
  }

  const data = res.status === 204 ? null : await res.json().catch(() => null);
  if (!res.ok) {
    if (res.status === 401 && token) onUnauthorized();
    throw new ApiError(res.status, data?.error ?? `Request failed (${res.status})`);
  }
  return data as T;
}

function query(params: Record<string, string | number | undefined>): string {
  const q = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== "") q.set(k, String(v));
  }
  const s = q.toString();
  return s ? `?${s}` : "";
}

export interface RegisterInput {
  email: string;
  password: string;
  name: string;
  preferredLanguage: string;
  readingLevel: string;
  interests: string[];
  prefersAudio: boolean;
}

export type ProfileInput = Pick<User, "name" | "preferredLanguage" | "readingLevel" | "interests" | "prefersAudio">;

export interface ItemSearch {
  q?: string;
  category?: string;
  level?: string;
  language?: string;
  sort?: string;
  page?: number;
}

export interface Correction {
  position: number;
  original: string;
  suggested: string;
}

export const api = {
  login: (email: string, password: string) =>
    request<{ token: string; user: User }>("POST", "/auth/login", { email, password }),
  register: (input: RegisterInput) => request<User>("POST", "/auth/register", input),

  me: () => request<User>("GET", "/users/me"),
  updateMe: (input: ProfileInput) => request<User>("PUT", "/users/me", input),
  history: (type: string, page: number) =>
    request<Page<HistoryEntry>>("GET", "/users/me/history" + query({ type, page, limit: 20 })),
  recommendations: (limit = 10) =>
    request<{ items: Recommendation[] }>("GET", "/users/me/recommendations" + query({ limit })),

  items: (s: ItemSearch) => request<Page<Item>>("GET", "/items" + query({ ...s, limit: 20 })),
  item: (id: string) => request<Item>("GET", `/items/${id}`),
  similar: (id: string, limit = 10) =>
    request<{ items: SimilarItem[] }>("GET", `/items/${id}/similar` + query({ limit })),

  interact: (itemId: string, type: string, value?: number, correction?: Correction) =>
    request<unknown>("POST", "/interactions", { itemId, type, value, correction }),
};
