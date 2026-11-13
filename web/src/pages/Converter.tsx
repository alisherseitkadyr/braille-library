import { useEffect, useState, type FormEvent } from "react";
import { api } from "../api";
import { decodeParagraphs, type BrailleLine } from "../braille";
import BrailleView from "../components/BrailleView";
import { ErrorMessage, Loading, PageHeading } from "../components/common";
import type { BrailleEntry, BrailleResult } from "../types";

const KINDS: { kind: BrailleEntry["kind"]; caption: string }[] = [
  { kind: "letter", caption: "Letters" },
  { kind: "punctuation", caption: "Punctuation" },
  { kind: "sign", caption: "Number sign (before each run of digits)" },
  { kind: "digit", caption: "Digits (after the number sign)" },
];

export default function Converter() {
  const [text, setText] = useState("");
  const [result, setResult] = useState<(BrailleResult & { lines: BrailleLine[] }) | null>(null);
  const [table, setTable] = useState<BrailleEntry[] | null>(null);
  const [status, setStatus] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    api.brailleTable().then(setTable).catch((e) => setError(e.message));
  }, []);

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError("");
    setStatus("");
    setBusy(true);
    try {
      const [res, t] = await Promise.all([api.brailleConvert(text), api.brailleTable()]);
      setResult({ ...res, lines: decodeParagraphs(res.braille, t) });
      const cells = Array.from(res.braille).filter((c) => c.trim()).length;
      const skipped = res.skipped.length
        ? ` ${res.skipped.length} ${res.skipped.length === 1 ? "character was" : "different characters were"} not converted.`
        : "";
      setStatus(`Converted: ${cells} Braille cells.${skipped}`);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  };

  const copy = async () => {
    if (!result) return;
    try {
      await navigator.clipboard.writeText(result.braille);
      setStatus("The Braille text is copied.");
    } catch {
      setError("Copying is not allowed in this browser. Select the Braille text and copy it yourself.");
    }
  };

  return (
    <>
      <PageHeading>Braille converter</PageHeading>
      <p>
        Converts Kazakh or Russian text to uncontracted Kazakh Braille (Unicode Braille cells). The Russian letters use
        the same cells. Letters are converted as lowercase, without a capital sign, because sources disagree on the
        capital sign. Spaces and line breaks stay; other characters are not converted and are listed.
      </p>

      <form onSubmit={onSubmit} className="form">
        <div className="field">
          <label htmlFor="text">Text</label>
          <textarea id="text" rows={5} value={text} onChange={(e) => setText(e.target.value)} />
        </div>
        <button type="submit" disabled={busy}>
          {busy ? "Converting…" : "Convert"}
        </button>
      </form>

      <section aria-labelledby="result-heading">
        <h2 id="result-heading">Braille</h2>
        <p role="status" className="success">{status}</p>
        <ErrorMessage error={error} />
        {!result && <p>Type a text and press Convert.</p>}
        {result && (
          <>
            <div className="reading braille-reading">
              {/* A new result starts fresh: no cell from the previous one is shown. */}
              <BrailleView key={result.braille} lines={result.lines} />
            </div>
            <div className="button-row">
              <button type="button" onClick={copy} disabled={!result.braille}>
                Copy Braille
              </button>
            </div>
            <h3>Not converted</h3>
            {result.skipped.length === 0 ? (
              <p>Nothing: every character was converted.</p>
            ) : (
              <ul className="skipped">
                {result.skipped.map((c) => (
                  <li key={c}>
                    <code>{c}</code> (U+{c.codePointAt(0)!.toString(16).toUpperCase().padStart(4, "0")})
                  </li>
                ))}
              </ul>
            )}
          </>
        )}
      </section>

      <section aria-labelledby="table-heading">
        <h2 id="table-heading">Braille table</h2>
        <p>Source: Wikipedia "Kazakh Braille", based on UNESCO "World Braille Usage".</p>
        {!table && !error && <Loading what="the table" />}
        {table &&
          KINDS.map(({ kind, caption }) => (
            <table key={kind} className="braille-table">
              <caption>{caption}</caption>
              <thead>
                <tr>
                  <th scope="col">Character</th>
                  <th scope="col">Braille</th>
                  <th scope="col">Dots</th>
                </tr>
              </thead>
              <tbody>
                {table
                  .filter((e) => e.kind === kind)
                  .map((e) => (
                    <tr key={e.kind + e.char}>
                      <th scope="row" lang={kind === "letter" ? "kk" : undefined}>{e.char}</th>
                      <td className="braille-glyph" aria-hidden="true">{e.braille}</td>
                      <td>{e.dots}</td>
                    </tr>
                  ))}
              </tbody>
            </table>
          ))}
      </section>
    </>
  );
}
