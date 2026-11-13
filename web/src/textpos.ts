// Positions in an item's content are counted in characters (Unicode code
// points), as the API does, not in JavaScript UTF-16 units.

export function charLength(s: string): number {
  return Array.from(s).length;
}

export interface Paragraph {
  text: string;
  start: number; // character position of the paragraph in the content
}

/** Splits content into paragraphs (separated by a blank line) and remembers where each starts. */
export function paragraphs(content: string): Paragraph[] {
  const result: Paragraph[] = [];
  let start = 0;
  for (const text of content.split("\n\n")) {
    result.push({ text, start });
    start += charLength(text) + 2;
  }
  return result;
}

const wordChar = /[\p{L}\p{M}\p{N}-]/u;

/**
 * Trims spaces and punctuation at the edges of [start, end) (UTF-16 indexes
 * in text), then completes words that are only partly selected. Trimming comes
 * first: a double-click on Windows also selects the space after the word, and
 * that must not pull in the next word. Returns null if nothing is left.
 */
export function expandToWords(text: string, start: number, end: number): [number, number] | null {
  while (start < end && !wordChar.test(text[start])) start++;
  while (end > start && !wordChar.test(text[end - 1])) end--;
  if (start >= end) return null;
  while (start > 0 && wordChar.test(text[start - 1])) start--;
  while (end < text.length && wordChar.test(text[end])) end++;
  return [start, end];
}

export interface Occurrence {
  position: number; // character position in the content
  context: string; // a few words around it, for choosing between occurrences
}

/** All places where s occurs in content. */
export function occurrences(content: string, s: string): Occurrence[] {
  const result: Occurrence[] = [];
  if (!s) return result;
  for (let i = content.indexOf(s); i !== -1; i = content.indexOf(s, i + 1)) {
    const before = content.slice(Math.max(0, i - 25), i).replace(/\s+/g, " ");
    const after = content.slice(i + s.length, i + s.length + 25).replace(/\s+/g, " ");
    result.push({ position: charLength(content.slice(0, i)), context: `…${before}[${s}]${after}…` });
  }
  return result;
}
