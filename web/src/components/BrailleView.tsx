import { useRef, useState, type KeyboardEvent } from "react";
import type { BrailleLine, Cell } from "../braille";
import type { Paragraph } from "../textpos";

/** The six dots of a cell as a small picture: dots 1-2-3 on the left, 4-5-6 on the right. */
function DotPicture({ dots }: { dots: string }) {
  const raised = new Set(dots.split("-"));
  return (
    <span className="dot-picture">
      {["1", "4", "2", "5", "3", "6"].map((d) => (
        <span key={d} className={raised.has(d) ? "dot raised" : "dot"} />
      ))}
    </span>
  );
}

/**
 * Braille cells, with the text next to them when texts is given (side by side).
 * All cells are one stop for the Tab key; the arrow keys, Home and End move between
 * cells. Focusing or clicking a cell shows what it stands for and its dots.
 * The glyphs are hidden from screen readers: a cell is announced by its meaning.
 */
export default function BrailleView({
  lines,
  texts,
  lang,
}: {
  lines: BrailleLine[];
  texts?: Paragraph[];
  lang?: string;
}) {
  const [active, setActive] = useState(0);
  const [shown, setShown] = useState<Cell | null>(null);
  const container = useRef<HTMLDivElement>(null);

  const total = lines.reduce((n, line) => n + line.filter(Boolean).length, 0);

  const onKeyDown = (e: KeyboardEvent) => {
    const target = e.target as HTMLElement;
    if (!target.matches("button.cell")) return;
    const i = Number(target.dataset.i);
    const next = {
      ArrowRight: i + 1,
      ArrowDown: i + 1,
      ArrowLeft: i - 1,
      ArrowUp: i - 1,
      Home: 0,
      End: total - 1,
    }[e.key];
    if (next === undefined) return;
    e.preventDefault();
    const to = Math.max(0, Math.min(total - 1, next));
    setActive(to);
    container.current?.querySelectorAll<HTMLButtonElement>("button.cell")[to]?.focus();
  };

  let index = 0;
  return (
    <div className="braille-view" ref={container} onKeyDown={onKeyDown}>
      <p className="sr-only">
        The Braille cells below are shown for sighted readers. If you read Braille, your Braille display shows the
        text itself; choose "Text" to read it.
      </p>
      <div className="cell-info" aria-hidden="true">
        {shown ? (
          <>
            <span className="cell-info-glyph">{shown.glyph}</span>
            <DotPicture dots={shown.dots} />
            <span>
              <strong>{shown.name}</strong> — dots {shown.dots}
            </span>
          </>
        ) : (
          <span>Click a Braille cell, or press Tab to reach the cells and move with the arrow keys, to see its letter and dots.</span>
        )}
      </div>
      {lines.map((line, p) => (
        <div key={p} className={texts ? "side-row" : undefined}>
          {texts && (
            <p data-start={texts[p].start} lang={lang}>
              {texts[p].text}
            </p>
          )}
          <p className="braille-line" role="group" aria-label={texts ? undefined : `Braille paragraph ${p + 1}`}>
            {line.map((cell, k) => {
              if (!cell) return <span key={k} className="braille-space" aria-hidden="true">{" "}</span>;
              const i = index++;
              return (
                <button
                  key={k}
                  type="button"
                  className="cell"
                  data-i={i}
                  tabIndex={i === active ? 0 : -1}
                  aria-label={`${cell.name}, dots ${cell.dots}`}
                  onFocus={() => {
                    setActive(i);
                    setShown(cell);
                  }}
                  onClick={() => setShown(cell)}
                >
                  <span aria-hidden="true">{cell.glyph}</span>
                </button>
              );
            })}
          </p>
        </div>
      ))}
    </div>
  );
}
