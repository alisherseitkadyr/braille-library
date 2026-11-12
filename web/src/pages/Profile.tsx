import { useState, type FormEvent } from "react";
import { api } from "../api";
import { useAuth } from "../auth";
import { ErrorMessage, PageHeading } from "../components/common";
import ProfileFields, { type ProfileValues } from "../components/ProfileFields";

export default function Profile() {
  const { user, setUser } = useAuth();
  // RequireAuth guarantees a user here.
  const [name, setName] = useState(user!.name);
  const [profile, setProfile] = useState<ProfileValues>({
    preferredLanguage: user!.preferredLanguage,
    readingLevel: user!.readingLevel,
    interests: user!.interests,
    prefersAudio: user!.prefersAudio,
  });
  const [status, setStatus] = useState("");
  const [error, setError] = useState("");

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setStatus("");
    setError("");
    try {
      setUser(await api.updateMe({ name, ...profile }));
      setStatus("Your profile is saved.");
    } catch (err) {
      setError((err as Error).message);
    }
  };

  return (
    <>
      <PageHeading>Profile</PageHeading>
      <p>Email: {user!.email}</p>
      <form onSubmit={onSubmit} className="form">
        <p role="status" className="success">{status}</p>
        <ErrorMessage error={error} />
        <div className="field">
          <label htmlFor="name">Name</label>
          <input id="name" autoComplete="name" required value={name} onChange={(e) => setName(e.target.value)} />
        </div>
        <ProfileFields values={profile} onChange={setProfile} />
        <button type="submit">Save profile</button>
      </form>
    </>
  );
}
