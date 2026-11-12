import { useEffect, useRef, useState, type FormEvent } from "react";
import { Link, useParams } from "react-router-dom";
import { api } from "../api";
import { useAuth } from "../auth";
import { ErrorMessage, ItemCard, Loading, PageHeading, itemMeta, ratingText } from "../components/common";
import type { Item, SimilarItem } from "../types";

export default function ItemPage() {
  const { id = "" } = useParams();
  const { user } = useAuth();
  const [item, setItem] = useState<Item | null>(null);
  const [similar, setSimilar] = useState<SimilarItem[] | null>(null);
  const [error, setError] = useState("");
  const viewed = useRef("");

  useEffect(() => {
    setItem(null);
    setSimilar(null);
    setError("");
    api.item(id).then(setItem).catch((e) => setError(e.message));
    api.similar(id, 10).then((r) => setSimilar(r.items)).catch(() => setSimilar([]));
  }, [id]);

  // Opening a text counts as a view (once per page visit).
  useEffect(() => {
    if (user && item && viewed.current !== item.id) {
      viewed.current = item.id;
      api.interact(item.id, "view").catch(() => {});
    }
  }, [user, item]);

  if (error) {
    return (
      <>
        <PageHeading>Text not found</PageHeading>
        <ErrorMessage error={error} />
        <p><Link to="/catalog">Back to the catalog</Link></p>
      </>
    );
  }
  if (!item) return <Loading what="the text" />;

  return (
    <>
      <PageHeading lang={item.language}>{item.title}</PageHeading>
      <p lang={item.language}>{item.description}</p>
      <dl className="details">
        <dt>Author</dt>
        <dd>{item.author || "Unknown"}</dd>
        <dt>About</dt>
        <dd>{itemMeta(item)}</dd>
        <dt>Pages</dt>
        <dd>{item.pages}</dd>
        <dt>Rating</dt>
        <dd>{ratingText(item)}</dd>
        <dt>Recognition accuracy</dt>
        <dd>{Math.round(item.recognitionAccuracy * 1000) / 10}%</dd>
        {item.tags.length > 0 && (
          <>
            <dt>Tags</dt>
            <dd>{item.tags.join(", ")}</dd>
          </>
        )}
      </dl>

      {user ? (
        <Actions item={item} onRated={() => api.item(id).then(setItem)} />
      ) : (
        <p>
          <Link to="/login">Log in</Link> to rate this text, save it or report errors.
        </p>
      )}

      <section aria-labelledby="similar-heading">
        <h2 id="similar-heading">Similar texts</h2>
        {!similar && <Loading what="similar texts" />}
        {similar && similar.length === 0 && <p>No similar texts yet.</p>}
        {similar && similar.length > 0 && (
          <ul className="card-list">
            {similar.map((s) => (
              <li key={s.item.id}>
                <ItemCard item={s.item} headingLevel={3} />
              </li>
            ))}
          </ul>
        )}
      </section>
    </>
  );
}

function Actions({ item, onRated }: { item: Item; onRated: () => void }) {
  const [status, setStatus] = useState("");
  const [error, setError] = useState("");
  const [rating, setRating] = useState("");
  const [progress, setProgress] = useState("100");
  const [correction, setCorrection] = useState({ position: "0", original: "", suggested: "" });

  const send = async (action: () => Promise<unknown>, done: string) => {
    setStatus("");
    setError("");
    try {
      await action();
      setStatus(done);
      return true;
    } catch (e) {
      setError((e as Error).message);
      return false;
    }
  };

  const onRate = async (e: FormEvent) => {
    e.preventDefault();
    if (!rating) return setError("Choose a rating from 1 to 5.");
    if (await send(() => api.interact(item.id, "rate", Number(rating)), `Your rating ${rating} of 5 is saved.`)) onRated();
  };

  const onProgress = (e: FormEvent) => {
    e.preventDefault();
    send(() => api.interact(item.id, "read", Number(progress)), `Reading progress ${progress}% is saved.`);
  };

  const onCorrection = async (e: FormEvent) => {
    e.preventDefault();
    const ok = await send(
      () =>
        api.interact(item.id, "correction", undefined, {
          position: Number(correction.position),
          original: correction.original,
          suggested: correction.suggested,
        }),
      "Thank you! The recognition error is reported.",
    );
    if (ok) setCorrection({ position: "0", original: "", suggested: "" });
  };

  return (
    <section aria-labelledby="actions-heading" className="actions">
      <h2 id="actions-heading">Your actions</h2>
      {/* One live region for the results of all actions below. */}
      <p role="status" className="success">{status}</p>
      <ErrorMessage error={error} />

      <form onSubmit={onRate} className="form">
        <fieldset>
          <legend>Rate this text</legend>
          <div className="rating">
            {[1, 2, 3, 4, 5].map((v) => (
              <div key={v} className="check">
                <input type="radio" name="rating" id={`rating-${v}`} value={v}
                  checked={rating === String(v)} onChange={(e) => setRating(e.target.value)} />
                <label htmlFor={`rating-${v}`}>{v}</label>
              </div>
            ))}
          </div>
        </fieldset>
        <button type="submit">Save rating</button>
      </form>

      <div className="button-row">
        <button type="button" onClick={() => send(() => api.interact(item.id, "favorite"), "Added to your favorites.")}>
          Add to favorites
        </button>
        {item.hasAudio && (
          <button type="button" onClick={() => send(() => api.interact(item.id, "listen"), "Listening is recorded.")}>
            Listen to audio
          </button>
        )}
      </div>

      <form onSubmit={onProgress} className="form">
        <div className="field">
          <label htmlFor="progress">Reading progress, percent</label>
          <input id="progress" type="number" min={0} max={100} required value={progress}
            onChange={(e) => setProgress(e.target.value)} />
        </div>
        <button type="submit">Save progress</button>
      </form>

      <form onSubmit={onCorrection} className="form">
        <fieldset>
          <legend>Report a recognition error</legend>
          <div className="field">
            <label htmlFor="position">Position in the text (character number)</label>
            <input id="position" type="number" min={0} required value={correction.position}
              onChange={(e) => setCorrection({ ...correction, position: e.target.value })} />
          </div>
          <div className="field">
            <label htmlFor="original">Recognized text</label>
            <input id="original" required value={correction.original}
              onChange={(e) => setCorrection({ ...correction, original: e.target.value })} />
          </div>
          <div className="field">
            <label htmlFor="suggested">Correct text</label>
            <input id="suggested" required value={correction.suggested}
              onChange={(e) => setCorrection({ ...correction, suggested: e.target.value })} />
          </div>
        </fieldset>
        <button type="submit">Send report</button>
      </form>
    </section>
  );
}
