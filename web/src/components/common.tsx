import { useEffect, useRef } from "react";
import { Link } from "react-router-dom";
import { CATEGORIES, LANGUAGES, LEVELS, label, type Item } from "../types";

/** Sets the browser tab title; the visible <h1> is rendered by the page. */
export function useTitle(title: string) {
  useEffect(() => {
    document.title = `${title} — Braille Library`;
  }, [title]);
}

// Set by the layout after the first navigation. On the very first page load
// focus stays at the top of the page, so the skip link is the first Tab stop.
export const navigation = { happened: false };

export function PageHeading({ children, lang }: { children: string; lang?: string }) {
  useTitle(children);
  const ref = useRef<HTMLHeadingElement>(null);
  // A page that shows "Loading…" first renders its heading later than the
  // navigation, so the heading focuses itself when it appears.
  useEffect(() => {
    if (navigation.happened) ref.current?.focus();
  }, []);
  return (
    <h1 ref={ref} tabIndex={-1} lang={lang}>
      {children}
    </h1>
  );
}

export function ErrorMessage({ error }: { error: string }) {
  if (!error) return null;
  return (
    <p role="alert" className="error">
      {error}
    </p>
  );
}

export function Loading({ what }: { what: string }) {
  return <p role="status">Loading {what}…</p>;
}

export function itemMeta(item: Item): string {
  const parts = [
    label(CATEGORIES, item.category),
    label(LEVELS, item.level),
    label(LANGUAGES, item.language),
    `Braille grade ${item.brailleGrade}`,
  ];
  if (item.hasAudio) parts.push("has audio");
  return parts.join(" · ");
}

export function ratingText(item: Item): string {
  if (item.ratingsCount === 0) return "No ratings yet";
  return `Rating ${item.avgRating.toFixed(1)} of 5 (${item.ratingsCount} ${item.ratingsCount === 1 ? "rating" : "ratings"})`;
}

/** One item in a list. headingLevel keeps the heading outline correct on each page. */
export function ItemCard({
  item,
  note,
  headingLevel = 2,
}: {
  item: Item;
  note?: string;
  headingLevel?: 2 | 3;
}) {
  const Heading = headingLevel === 2 ? "h2" : "h3";
  return (
    <article className="card">
      <Heading className="card-title">
        <Link to={`/items/${item.id}`} lang={item.language}>
          {item.title}
        </Link>
      </Heading>
      {note && <p className="reason">{note}</p>}
      <p className="meta">{itemMeta(item)}</p>
      <p className="meta">
        {item.author && <>{item.author} · </>}
        {ratingText(item)}
      </p>
    </article>
  );
}

export function Pagination({
  page,
  total,
  limit,
  onPage,
}: {
  page: number;
  total: number;
  limit: number;
  onPage: (p: number) => void;
}) {
  const pages = Math.max(1, Math.ceil(total / limit));
  if (pages === 1) return null;
  return (
    <nav aria-label="Pages" className="pagination">
      <button type="button" disabled={page <= 1} onClick={() => onPage(page - 1)}>
        Previous page
      </button>
      <span>
        Page {page} of {pages}
      </span>
      <button type="button" disabled={page >= pages} onClick={() => onPage(page + 1)}>
        Next page
      </button>
    </nav>
  );
}
