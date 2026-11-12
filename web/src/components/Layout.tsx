import { useEffect, useRef, useState } from "react";
import { NavLink, Outlet, useLocation, useNavigate } from "react-router-dom";
import { useAuth } from "../auth";
import { navigation } from "./common";

const SIZES = [
  { value: "100%", label: "A", name: "Normal text" },
  { value: "125%", label: "A+", name: "Large text" },
  { value: "150%", label: "A++", name: "Extra large text" },
];
const SIZE_KEY = "fontSize";

export default function Layout() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const [size, setSize] = useState(() => localStorage.getItem(SIZE_KEY) ?? SIZES[0].value);
  const firstRender = useRef(true);

  useEffect(() => {
    document.documentElement.style.fontSize = size;
    localStorage.setItem(SIZE_KEY, size);
  }, [size]);

  // After navigation, move focus to the new page heading so screen readers
  // announce it (a single-page app does not reload the page).
  useEffect(() => {
    if (firstRender.current) {
      firstRender.current = false;
      return;
    }
    navigation.happened = true;
    document.querySelector<HTMLElement>("main h1")?.focus();
  }, [location.pathname]);

  const onLogout = () => {
    logout();
    navigate("/login");
  };

  return (
    <>
      <a className="skip-link" href="#main">
        Skip to main content
      </a>
      <header className="site-header">
        <div className="container header-row">
          <p className="site-name">
            <NavLink to="/">Braille Library</NavLink>
          </p>
          <div className="text-size" role="group" aria-label="Text size">
            {SIZES.map((s) => (
              <button
                key={s.value}
                type="button"
                aria-pressed={size === s.value}
                aria-label={s.name}
                onClick={() => setSize(s.value)}
              >
                {s.label}
              </button>
            ))}
          </div>
        </div>
        <nav aria-label="Main" className="container">
          <ul className="nav-list">
            {user ? (
              <>
                <li>
                  <NavLink to="/" end>
                    Recommended
                  </NavLink>
                </li>
                <li>
                  <NavLink to="/catalog">Catalog</NavLink>
                </li>
                <li>
                  <NavLink to="/history">History</NavLink>
                </li>
                <li>
                  <NavLink to="/profile">Profile</NavLink>
                </li>
                <li>
                  <button type="button" className="link-button" onClick={onLogout}>
                    Log out ({user.name})
                  </button>
                </li>
              </>
            ) : (
              <>
                <li>
                  <NavLink to="/catalog">Catalog</NavLink>
                </li>
                <li>
                  <NavLink to="/login">Log in</NavLink>
                </li>
                <li>
                  <NavLink to="/register">Register</NavLink>
                </li>
              </>
            )}
          </ul>
        </nav>
      </header>
      <main id="main" tabIndex={-1} className="container">
        <Outlet />
      </main>
    </>
  );
}
