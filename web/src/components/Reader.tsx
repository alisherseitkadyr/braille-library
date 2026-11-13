import { useEffect, useMemo, useRef } from "react";
import { expandToWords, charLength, paragraphs } from "../textpos";

export const PROGRESS_STEPS = [25, 50, 80, 100];

export interface PickedText {
  original: string;
  position: number; // character position in the content
}

/**
 * The text of an item. While the reader scrolls, onProgress is called with
 * 25, 50, 80 and 100 (the share of the text above the bottom of the window),
 * each step at most once. Selecting text calls onPick with the selection
 * expanded to whole words and its character position in the content.
 */
export default function Reader({
  content,
  lang,
  onProgress,
  onPick,
}: {
  content: string;
  lang: string;
  onProgress: (step: number) => void;
  onPick: (picked: PickedText) => void;
}) {
  const textRef = useRef<HTMLDivElement>(null);
  const reached = useRef(0); // highest step already reported for this text
  const report = useRef(onProgress);
  report.current = onProgress;
  const parts = useMemo(() => paragraphs(content), [content]);

  useEffect(() => {
    reached.current = 0;
    let frame = 0;
    // Only real scrolling counts (also with the keyboard: arrows, Page Down,
    // Space), so a page that is merely opened records nothing.
    const onScroll = () => {
      cancelAnimationFrame(frame);
      frame = requestAnimationFrame(() => {
        const el = textRef.current;
        if (!el) return;
        const rect = el.getBoundingClientRect();
        // Count only while some of the text is on screen: jumping past it
        // (End key, a link to the bottom of the page) is not reading.
        if (rect.bottom <= 0 || rect.top >= window.innerHeight) return;
        const percent = ((window.innerHeight - rect.top) / rect.height) * 100;
        const step = PROGRESS_STEPS.filter((s) => percent >= s).pop();
        if (step && step > reached.current) {
          reached.current = step;
          report.current(step);
        }
      });
    };
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => {
      window.removeEventListener("scroll", onScroll);
      cancelAnimationFrame(frame);
    };
  }, [content]);

  const pickSelection = () => {
    const sel = window.getSelection();
    if (!sel || sel.isCollapsed || sel.rangeCount === 0) return;
    const range = sel.getRangeAt(0);
    const node = range.startContainer;
    const p = node.parentElement?.closest<HTMLElement>("p[data-start]");
    // One paragraph is one text node; selections across paragraphs are ignored.
    if (node.nodeType !== Node.TEXT_NODE || !p || !textRef.current?.contains(p) || range.endContainer !== node) {
      return;
    }
    const text = node.textContent ?? "";
    const words = expandToWords(text, range.startOffset, range.endOffset);
    if (!words) return;
    const [start, end] = words;
    onPick({
      original: text.slice(start, end),
      position: Number(p.dataset.start) + charLength(text.slice(0, start)),
    });
  };

  return (
    <div ref={textRef} className="reading" lang={lang} onMouseUp={pickSelection} onKeyUp={pickSelection}>
      {parts.map((part) => (
        <p key={part.start} data-start={part.start}>
          {part.text}
        </p>
      ))}
    </div>
  );
}
