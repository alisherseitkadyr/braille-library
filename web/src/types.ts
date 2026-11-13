export type Category = "fiction" | "textbook" | "children" | "news" | "exercises";
export type Level = "beginner" | "intermediate" | "advanced";
export type Language = "kk" | "ru";
export type InteractionType = "view" | "read" | "listen" | "rate" | "favorite" | "correction";

export const CATEGORIES: { value: Category; label: string }[] = [
  { value: "fiction", label: "Fiction" },
  { value: "textbook", label: "Textbook" },
  { value: "children", label: "Children" },
  { value: "news", label: "News" },
  { value: "exercises", label: "Exercises" },
];

export const LEVELS: { value: Level; label: string }[] = [
  { value: "beginner", label: "Beginner" },
  { value: "intermediate", label: "Intermediate" },
  { value: "advanced", label: "Advanced" },
];

export const LANGUAGES: { value: Language; label: string }[] = [
  { value: "kk", label: "Kazakh" },
  { value: "ru", label: "Russian" },
];

export const INTERACTION_LABELS: Record<InteractionType, string> = {
  view: "Viewed",
  read: "Read",
  listen: "Listened",
  rate: "Rated",
  favorite: "Added to favorites",
  correction: "Reported a recognition error",
};

export function label<T extends string>(options: { value: T; label: string }[], value: T): string {
  return options.find((o) => o.value === value)?.label ?? value;
}

export interface User {
  id: string;
  email: string;
  name: string;
  role: string;
  preferredLanguage: Language;
  readingLevel: Level;
  interests: Category[];
  prefersAudio: boolean;
}

export interface Item {
  id: string;
  title: string;
  description: string;
  category: Category;
  level: Level;
  language: Language;
  brailleGrade: number;
  pages: number;
  author: string;
  tags: string[];
  hasAudio: boolean;
  recognitionAccuracy: number;
  avgRating: number;
  ratingsCount: number;
  content?: string; // only in GET /items/{id}
}

export interface Page<T> {
  items: T[];
  page: number;
  limit: number;
  total: number;
}

export interface Recommendation {
  item: Item;
  score: number;
  reason: string;
}

export interface SimilarItem {
  item: Item;
  score: number;
}

export interface HistoryEntry {
  id: string;
  itemId: string;
  itemTitle: string;
  type: InteractionType;
  value?: number;
  correction?: { position: number; original: string; suggested: string };
  createdAt: string;
}

export interface BrailleResult {
  braille: string;
  skipped: string[];
}

export interface BrailleEntry {
  char: string;
  braille: string;
  dots: string;
  kind: "letter" | "punctuation" | "digit" | "sign";
}
