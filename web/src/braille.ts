import type { BrailleEntry } from "./types";

export interface Cell {
  glyph: string; // the Braille pattern, e.g. ⠹
  name: string; // what it stands for: "қ", "comma", "digit 2", "number sign"
  dots: string; // e.g. "1-4-5-6"
}

/** A line of Braille: cells and spaces (null). */
export type BrailleLine = (Cell | null)[];

const punctuationNames: Record<string, string> = {
  ",": "comma",
  ".": "period",
  "?": "question mark",
  "!": "exclamation mark",
  ";": "semicolon",
  ":": "colon",
};

/**
 * Reads Braille from the API back into cells with their meaning. Letter and
 * punctuation cells are unique, so a cell names one character; after the number
 * sign, cells are digits until anything else comes (the same rule the converter uses).
 */
export function decode(braille: string, table: BrailleEntry[]): BrailleLine {
  const letters = new Map(table.filter((e) => e.kind === "letter" || e.kind === "punctuation").map((e) => [e.braille, e]));
  const digits = new Map(table.filter((e) => e.kind === "digit").map((e) => [e.braille, e]));
  const sign = table.find((e) => e.kind === "sign");

  const line: BrailleLine = [];
  let inNumber = false;
  for (const glyph of Array.from(braille)) {
    if (sign && glyph === sign.braille) {
      line.push({ glyph, name: "number sign", dots: sign.dots });
      inNumber = true;
      continue;
    }
    const digit = inNumber ? digits.get(glyph) : undefined;
    if (digit) {
      line.push({ glyph, name: `digit ${digit.char}`, dots: digit.dots });
      continue;
    }
    inNumber = false;
    const entry = letters.get(glyph);
    if (entry) {
      line.push({ glyph, name: punctuationNames[entry.char] ?? entry.char, dots: entry.dots });
    } else {
      line.push(null); // a space or a line break
    }
  }
  return line;
}

/** Braille text split into paragraphs (blank line), like the item content. */
export function decodeParagraphs(braille: string, table: BrailleEntry[]): BrailleLine[] {
  return braille.split("\n\n").map((p) => decode(p.replace(/\n/g, " "), table));
}
