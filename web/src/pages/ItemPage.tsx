import { useCallback, useEffect, useMemo, useRef, useState, type FormEvent } from "react";
import { Link, useParams } from "react-router-dom";
import { api } from "../api";
import { useAuth } from "../auth";
import { ErrorMessage, ItemCard, Loading, PageHeading, itemMeta, ratingText } from "../components/common";
import Listen from "../components/Listen";
import Reader, { type PickedText } from "../components/Reader";
import { occurrences } from "../textpos";
import type { Item, SimilarItem } from "../types";

export default function ItemPage() {
  const { id = "" } = useParams();
  const { user } = useAuth();
  const [item, setItem] = useState<Item | null>(null);
  const [similar, setSimilar] = useState<SimilarItem[] | null>(null);
  const [error, setError] = useState("");
  const [readStatus, setReadStatus] = useState("");
  const [picked, setPicked] = useState<PickedText | null>(null);
  const viewed = useRef("");

  useEffect(() => {
    setItem(null);
    setSimilar(null);
    setError("");
    setReadStatus("");
    setPicked(null);
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

  const loggedIn = Boolean(user);
  const onProgress = useCallback(
    (step: number) => {
      if (!loggedIn) return;
      api
        .interact(id, "read", step)
        .then(() => setReadStatus(`Reading progress ${step}% saved.`))
        .catch(() => {});
    },
    [id, loggedIn],
  );
  const onListen = useCallback(() => {
    if (loggedIn) api.interact(id, "listen").catch(() => {});
  }, [id, loggedIn]);

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

      {item.content ? (
        <>
          <section aria-labelledby="listen-heading">
            <h2 id="listen-heading">Listen</h2>
            <Listen content={item.content} lang={item.language} onFirstPlay={onListen} />
          </section>

          <section aria-labelledby="read-heading">
            <h2 id="read-heading">Read</h2>
            <p className="hint">
              {user
                ? "Your reading progress is saved as you scroll. To report a recognition error, select the wrong word in the text, or type it in the form below the text."
                : "Log in to save your reading progress and report recognition errors."}
            </p>
            <p role="status" className="success">{readStatus}</p>
            <Reader content={item.content} lang={item.language} onProgress={onProgress} onPick={setPicked} />
          </section>
        </>
      ) : (
        <p>The text is not available yet.</p>
      )}

      {user ? (
        <Actions item={item} picked={picked} onRated={() => api.item(id).then(setItem)} />
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

function Actions({ item, picked, onRated }: { item: Item; picked: PickedText | null; onRated: () => void }) {
  const [status, setStatus] = useState("");
  const [error, setError] = useState("");
  const [rating, setRating] = useState("");
  const [progress, setProgress] = useState("100");
  const [original, setOriginal] = useState("");
  const [occurrence, setOccurrence] = useState(0);
  const [suggested, setSuggested] = useState("");
  const suggestedInput = useRef<HTMLInputElement>(null);

  const content = item.content ?? "";
  const found = useMemo(() => occurrences(content, original), [content, original]);

  // A word selected in the text fills the form; the user then types the correction.
  useEffect(() => {
    if (!picked) return;
    setOriginal(picked.original);
    const all = occurrences(content, picked.original);
    setOccurrence(Math.max(0, all.findIndex((o) => o.position === picked.position)));
    setError("");
    setStatus(`Selected "${picked.original}". Type the correct text in the form "Report a recognition error".`);
    suggestedInput.current?.focus();
  }, [picked, content]);

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
    const at = found[occurrence];
    if (!at) {
      setStatus("");
      return setError("The recognized text was not found in this text. Select it in the text or check the spelling.");
    }
    const ok = await send(
      () => api.interact(item.id, "correction", undefined, { position: at.position, original, suggested }),
      `Thank you! The error "${original}" → "${suggested}" is reported.`,
    );
    if (ok) {
      setOriginal("");
      setSuggested("");
      setOccurrence(0);
    }
  };

  let where = "";
  if (original) {
    if (found.length === 0) where = "Not found in the text.";
    else if (found.length === 1) where = `Found at character ${found[0].position}.`;
    else where = `Found ${found.length} times; choose which one below.`;
  }

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
      </div>

      <form onSubmit={onProgress} className="form">
        <div className="field">
          <label htmlFor="progress">Reading progress, percent</label>
          <input id="progress" type="number" min={0} max={100} required value={progress}
            aria-describedby="progress-hint" onChange={(e) => setProgress(e.target.value)} />
          <p id="progress-hint" className="hint">Saved automatically while you scroll the text; use this if you read it elsewhere.</p>
        </div>
        <button type="submit">Save progress</button>
      </form>

      {content && (
        <form onSubmit={onCorrection} className="form">
          <fieldset>
            <legend>Report a recognition error</legend>
            <p id="correction-hint" className="hint">
              Select the wrong word in the text above, or type it here exactly as it appears.
            </p>
            <div className="field">
              <label htmlFor="original">Recognized text (wrong)</label>
              <input id="original" required value={original} lang={item.language}
                aria-describedby="correction-hint original-where"
                onChange={(e) => { setOriginal(e.target.value); setOccurrence(0); }} />
              <p id="original-where" className="hint">{where}</p>
            </div>
            {found.length > 1 && (
              <div className="field">
                <label htmlFor="occurrence">Which one?</label>
                <select id="occurrence" value={occurrence} lang={item.language}
                  onChange={(e) => setOccurrence(Number(e.target.value))}>
                  {found.map((o, i) => (
                    <option key={o.position} value={i}>
                      {i + 1}. {o.context}
                    </option>
                  ))}
                </select>
              </div>
            )}
            <div className="field">
              <label htmlFor="suggested">Correct text</label>
              <input id="suggested" ref={suggestedInput} required value={suggested} lang={item.language}
                onChange={(e) => setSuggested(e.target.value)} />
            </div>
          </fieldset>
          <button type="submit">Send report</button>
        </form>
      )}
    </section>
  );
}
