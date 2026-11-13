import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { api } from "../api";
import { ErrorMessage, Loading, PageHeading, Pagination } from "../components/common";
import { INTERACTION_LABELS, type HistoryEntry, type InteractionType, type Page } from "../types";

const dateFormat = new Intl.DateTimeFormat("en-GB", { dateStyle: "medium", timeStyle: "short" });

function describe(e: HistoryEntry): string {
  let text = INTERACTION_LABELS[e.type];
  if (e.type === "rate") text += ` ${e.value} of 5`;
  if (e.type === "read") text += `, progress ${e.value}%`;
  if (e.type === "correction" && e.correction) {
    text += ` "${e.correction.original}" → "${e.correction.suggested}"`;
  }
  return text;
}

export default function History() {
  const [type, setType] = useState("");
  const [page, setPage] = useState(1);
  const [result, setResult] = useState<Page<HistoryEntry> | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    setResult(null);
    setError("");
    api.history(type, page).then(setResult).catch((e) => setError(e.message));
  }, [type, page]);

  return (
    <>
      <PageHeading>History</PageHeading>
      <div className="field">
        <label htmlFor="type">Show</label>
        <select id="type" value={type} onChange={(e) => { setType(e.target.value); setPage(1); }}>
          <option value="">All actions</option>
          {(Object.keys(INTERACTION_LABELS) as InteractionType[]).map((t) => (
            <option key={t} value={t}>{INTERACTION_LABELS[t]}</option>
          ))}
        </select>
      </div>

      <ErrorMessage error={error} />
      {!result && !error && <Loading what="history" />}
      {result && (
        <>
          <p role="status">
            {result.total} {result.total === 1 ? "action" : "actions"}
          </p>
          <ol className="history">
            {result.items.map((e) => (
              <li key={e.id}>
                <time dateTime={e.createdAt}>{dateFormat.format(new Date(e.createdAt))}</time>
                {" — "}
                {describe(e)}:{" "}
                {e.itemTitle ? <Link to={`/items/${e.itemId}`} lang="">{e.itemTitle}</Link> : "deleted text"}
              </li>
            ))}
          </ol>
          <Pagination page={page} total={result.total} limit={result.limit} onPage={setPage} />
        </>
      )}
    </>
  );
}
