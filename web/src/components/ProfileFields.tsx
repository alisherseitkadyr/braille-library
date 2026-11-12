import { CATEGORIES, LANGUAGES, LEVELS, type Category, type Language, type Level } from "../types";

export interface ProfileValues {
  preferredLanguage: Language;
  readingLevel: Level;
  interests: Category[];
  prefersAudio: boolean;
}

/** Reading preferences used by both registration and the profile page. */
export default function ProfileFields({
  values,
  onChange,
}: {
  values: ProfileValues;
  onChange: (v: ProfileValues) => void;
}) {
  const toggleInterest = (c: Category, checked: boolean) =>
    onChange({
      ...values,
      interests: checked ? [...values.interests, c] : values.interests.filter((x) => x !== c),
    });

  return (
    <>
      <div className="field">
        <label htmlFor="preferredLanguage">Preferred language</label>
        <select
          id="preferredLanguage"
          value={values.preferredLanguage}
          onChange={(e) => onChange({ ...values, preferredLanguage: e.target.value as Language })}
        >
          {LANGUAGES.map((l) => (
            <option key={l.value} value={l.value}>
              {l.label}
            </option>
          ))}
        </select>
      </div>

      <div className="field">
        <label htmlFor="readingLevel">Reading level</label>
        <select
          id="readingLevel"
          value={values.readingLevel}
          onChange={(e) => onChange({ ...values, readingLevel: e.target.value as Level })}
        >
          {LEVELS.map((l) => (
            <option key={l.value} value={l.value}>
              {l.label}
            </option>
          ))}
        </select>
      </div>

      <fieldset className="field">
        <legend>Interests</legend>
        {CATEGORIES.map((c) => (
          <div key={c.value} className="check">
            <input
              type="checkbox"
              id={`interest-${c.value}`}
              checked={values.interests.includes(c.value)}
              onChange={(e) => toggleInterest(c.value, e.target.checked)}
            />
            <label htmlFor={`interest-${c.value}`}>{c.label}</label>
          </div>
        ))}
      </fieldset>

      <div className="field check">
        <input
          type="checkbox"
          id="prefersAudio"
          checked={values.prefersAudio}
          onChange={(e) => onChange({ ...values, prefersAudio: e.target.checked })}
        />
        <label htmlFor="prefersAudio">I prefer texts with audio</label>
      </div>
    </>
  );
}
