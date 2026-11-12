import { useEffect, useState, type FormEvent } from "react";
import { useSearchParams } from "react-router-dom";
import { api } from "../api";
import { ErrorMessage, ItemCard, Loading, PageHeading, Pagination } from "../components/common";
import { CATEGORIES, LANGUAGES, LEVELS, type Item, type Page } from "../types";

const FIELDS = ["q", "category", "level", "language", "sort"] as const;
type Filters = Record<(typeof FIELDS)[number], string>;

export default function Catalog() {
  // The search lives in the URL, so it survives reload and the back button.
  const [params, setParams] = useSearchParams();
  const page = Number(params.get("page") ?? "1") || 1;
  const fromParams = (): Filters =>
    Object.fromEntries(FIELDS.map((f) => [f, params.get(f) ?? ""])) as Filters;

  const [filters, setFilters] = useState<Filters>(fromParams);
  const [result, setResult] = useState<Page<Item> | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    setResult(null);
    setError("");
    api
      .items({ ...fromParams(), page })
      .then(setResult)
      .catch((e) => setError(e.message));
  }, [params]);

  const search = (f: Filters, p: number) => {
    const next = new URLSearchParams();
    for (const key of FIELDS) if (f[key]) next.set(key, f[key]);
    if (p > 1) next.set("page", String(p));
    setParams(next);
  };

  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    search(filters, 1);
  };

  const set = (key: keyof Filters) => (e: { target: { value: string } }) =>
    setFilters({ ...filters, [key]: e.target.value });

  return (
    <>
      <PageHeading>Catalog</PageHeading>
      <form role="search" aria-label="Search the catalog" onSubmit={onSubmit} className="form filters">
        <div className="field">
          <label htmlFor="q">Search words</label>
          <input id="q" type="search" value={filters.q} onChange={set("q")} />
        </div>
        <div className="field">
          <label htmlFor="category">Category</label>
          <select id="category" value={filters.category} onChange={set("category")}>
            <option value="">Any category</option>
            {CATEGORIES.map((c) => (
              <option key={c.value} value={c.value}>{c.label}</option>
            ))}
          </select>
        </div>
        <div className="field">
          <label htmlFor="level">Level</label>
          <select id="level" value={filters.level} onChange={set("level")}>
            <option value="">Any level</option>
            {LEVELS.map((l) => (
              <option key={l.value} value={l.value}>{l.label}</option>
            ))}
          </select>
        </div>
        <div className="field">
          <label htmlFor="language">Language</label>
          <select id="language" value={filters.language} onChange={set("language")}>
            <option value="">Any language</option>
            {LANGUAGES.map((l) => (
              <option key={l.value} value={l.value}>{l.label}</option>
            ))}
          </select>
        </div>
        <div className="field">
          <label htmlFor="sort">Sort by</label>
          <select id="sort" value={filters.sort} onChange={set("sort")} aria-describedby="sort-hint">
            <option value="">Newest first</option>
            <option value="rating">Highest rated first</option>
          </select>
          <p id="sort-hint" className="hint">With search words, the best matches come first.</p>
        </div>
        <button type="submit">Search</button>
      </form>

      <section aria-labelledby="results-heading">
        <h2 id="results-heading">Results</h2>
        <ErrorMessage error={error} />
        {!result && !error && <Loading what="texts" />}
        {result && (
          <>
            <p role="status">
              {result.total} {result.total === 1 ? "text" : "texts"} found
            </p>
            <ul className="card-list">
              {result.items.map((item) => (
                <li key={item.id}>
                  <ItemCard item={item} headingLevel={3} />
                </li>
              ))}
            </ul>
            <Pagination page={page} total={result.total} limit={result.limit} onPage={(p) => search(fromParams(), p)} />
          </>
        )}
      </section>
    </>
  );
}
