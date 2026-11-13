import { useEffect, useRef, useState } from "react";
import { LANGUAGES, label, type Language } from "../types";
import { paragraphs } from "../textpos";

type State = "stopped" | "playing" | "paused";

const supported = typeof window !== "undefined" && "speechSynthesis" in window;

/** Picks a voice for the language ("kk", "ru"); null means the browser's default voice. */
function voiceFor(lang: string): SpeechSynthesisVoice | null {
  const voices = window.speechSynthesis.getVoices();
  return voices.find((v) => v.lang.toLowerCase().startsWith(lang)) ?? null;
}

/**
 * Reads the text aloud with the browser's speech synthesis. onFirstPlay is
 * called the first time playback starts (to record a "listen").
 */
export default function Listen({
  content,
  lang,
  onFirstPlay,
}: {
  content: string;
  lang: Language;
  onFirstPlay: () => void;
}) {
  const [state, setState] = useState<State>("stopped");
  const [note, setNote] = useState("");
  const [status, setStatus] = useState(""); // announced to screen readers
  const [error, setError] = useState("");
  const run = useRef(0); // ignores events of utterances from an earlier run
  const played = useRef(false);
  const playButton = useRef<HTMLButtonElement>(null);
  const stopButton = useRef<HTMLButtonElement>(null);

  // Stop speaking when the text changes or the page is left.
  useEffect(() => {
    played.current = false;
    return () => {
      run.current++;
      if (supported) window.speechSynthesis.cancel();
    };
  }, [content]);

  if (!supported) {
    return <p>Your browser cannot read texts aloud.</p>;
  }
  const synth = window.speechSynthesis;

  // Stop is disabled when nothing plays. If it has the focus, move the focus to
  // Play first, so a keyboard user does not lose their place.
  const keepFocus = () => {
    if (document.activeElement === stopButton.current) playButton.current?.focus();
  };

  const play = () => {
    setError("");
    if (state === "paused") {
      synth.resume();
      setState("playing");
      setStatus("Reading aloud");
      return;
    }

    const id = ++run.current;
    synth.cancel();
    const voice = voiceFor(lang);
    setNote(
      voice
        ? ""
        : `No ${label(LANGUAGES, lang)} voice is installed in this browser, so the default voice reads the text. It may mispronounce some words.`,
    );

    // One utterance per paragraph: some browsers stop very long utterances.
    const parts = paragraphs(content);
    parts.forEach((part, i) => {
      const u = new SpeechSynthesisUtterance(part.text);
      if (voice) {
        u.voice = voice;
        u.lang = voice.lang;
      }
      u.onerror = (e) => {
        if (id !== run.current || e.error === "interrupted" || e.error === "canceled") return;
        setError(`Reading aloud failed (${e.error}).`);
        keepFocus();
        setState("stopped");
        setStatus("");
      };
      if (i === parts.length - 1) {
        u.onend = () => {
          if (id !== run.current) return;
          keepFocus();
          setState("stopped");
          setStatus("Finished");
        };
      }
      synth.speak(u);
    });
    setState("playing");
    setStatus("Reading aloud");

    if (!played.current) {
      played.current = true;
      onFirstPlay();
    }
  };

  const pause = () => {
    synth.pause();
    setState("paused");
    setStatus("Paused");
  };

  const stop = () => {
    run.current++;
    synth.cancel();
    keepFocus();
    setState("stopped");
    setStatus("Stopped");
  };

  return (
    <div className="listen">
      <div className="button-row" role="group" aria-label="Read aloud" aria-describedby={note ? "voice-note" : undefined}>
        {/* One toggle button that is never disabled, so the keyboard focus stays on it. */}
        <button ref={playButton} type="button" onClick={state === "playing" ? pause : play}>
          {state === "playing" ? "Pause" : state === "paused" ? "Resume" : "Play"}
        </button>
        <button ref={stopButton} type="button" onClick={stop} disabled={state === "stopped"}>
          Stop
        </button>
      </div>
      <p role="status">{status}</p>
      {note && (
        <p id="voice-note" className="hint">
          {note}
        </p>
      )}
      {error && (
        <p role="alert" className="error">
          {error}
        </p>
      )}
    </div>
  );
}
