import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { api } from "../api";
import { ErrorMessage, ItemCard, Loading, PageHeading } from "../components/common";
import type { Recommendation } from "../types";

export default function Home() {
  const [recs, setRecs] = useState<Recommendation[] | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    api
      .recommendations(10)
      .then((r) => setRecs(r.items))
      .catch((e) => setError(e.message));
  }, []);

  return (
    <>
      <PageHeading>Recommended for you</PageHeading>
      <ErrorMessage error={error} />
      {!recs && !error && <Loading what="recommendations" />}
      {recs && recs.length === 0 && (
        <p>
          No recommendations yet. Browse the <Link to="/catalog">catalog</Link> to get started.
        </p>
      )}
      {recs && recs.length > 0 && (
        <ul className="card-list">
          {recs.map((r) => (
            <li key={r.item.id}>
              <ItemCard item={r.item} note={capitalize(r.reason)} />
            </li>
          ))}
        </ul>
      )}
    </>
  );
}

function capitalize(s: string): string {
  return s.charAt(0).toUpperCase() + s.slice(1);
}
