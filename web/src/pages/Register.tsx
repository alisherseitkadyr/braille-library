import { useState, type FormEvent } from "react";
import { Link, useNavigate } from "react-router-dom";
import { api } from "../api";
import { useAuth } from "../auth";
import { ErrorMessage, PageHeading } from "../components/common";
import ProfileFields, { type ProfileValues } from "../components/ProfileFields";

export default function Register() {
  const { login } = useAuth();
  const navigate = useNavigate();
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [profile, setProfile] = useState<ProfileValues>({
    preferredLanguage: "kk",
    readingLevel: "beginner",
    interests: [],
    prefersAudio: false,
  });
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError("");
    setBusy(true);
    try {
      await api.register({ name, email, password, ...profile });
      await login(email, password);
      navigate("/", { replace: true });
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  };

  return (
    <>
      <PageHeading>Register</PageHeading>
      <form onSubmit={onSubmit} className="form">
        <ErrorMessage error={error} />
        <div className="field">
          <label htmlFor="name">Name</label>
          <input id="name" autoComplete="name" required value={name} onChange={(e) => setName(e.target.value)} />
        </div>
        <div className="field">
          <label htmlFor="email">Email</label>
          <input id="email" type="email" autoComplete="email" required value={email}
            onChange={(e) => setEmail(e.target.value)} />
        </div>
        <div className="field">
          <label htmlFor="password">Password</label>
          <input id="password" type="password" autoComplete="new-password" required minLength={8}
            aria-describedby="password-hint" value={password} onChange={(e) => setPassword(e.target.value)} />
          <p id="password-hint" className="hint">At least 8 characters.</p>
        </div>
        <ProfileFields values={profile} onChange={setProfile} />
        <button type="submit" disabled={busy}>
          {busy ? "Creating account…" : "Create account"}
        </button>
      </form>
      <p>
        Already registered? <Link to="/login">Log in</Link>
      </p>
    </>
  );
}
