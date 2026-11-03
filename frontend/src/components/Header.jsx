import { useEffect, useRef, useState } from 'react';

export default function Header() {
  const [isOpen, setIsOpen] = useState(false);
  const headerRef = useRef(null);
  const toggleRef = useRef(null);

  useEffect(() => {
    const desktop = window.matchMedia('(min-width: 1101px)');
    const closeNavigation = () => setIsOpen(false);
    desktop.addEventListener('change', closeNavigation);

    return () => desktop.removeEventListener('change', closeNavigation);
  }, []);

  useEffect(() => {
    if (!isOpen) return;

    const handleKeyDown = (event) => {
      if (event.key === 'Escape') {
        setIsOpen(false);
        toggleRef.current?.focus();
      }
    };
    const handleOutsideClick = (event) => {
      if (!headerRef.current?.contains(event.target)) setIsOpen(false);
    };

    document.addEventListener('keydown', handleKeyDown);
    document.addEventListener('click', handleOutsideClick);
    return () => {
      document.removeEventListener('keydown', handleKeyDown);
      document.removeEventListener('click', handleOutsideClick);
    };
  }, [isOpen]);

  return (
    <header ref={headerRef} className={`navbar nav-enhanced${isOpen ? ' nav-open' : ''}`}>
      <a className="logo" href="/" aria-label="Agrilinks Nigeria home">
        <img
          src="/static/assets/logo/agrilinks-nigeria-logo.svg"
          alt="Agrilinks Nigeria logo"
          width="180"
          height="50"
        />
      </a>

      <button
        ref={toggleRef}
        type="button"
        className="nav-toggle"
        aria-controls="main-navigation"
        aria-expanded={isOpen}
        aria-label={isOpen ? 'Close navigation' : 'Open navigation'}
        onClick={() => setIsOpen((open) => !open)}
      >
        <span aria-hidden="true">{isOpen ? '×' : '☰'}</span>
      </button>

      <nav id="main-navigation" aria-label="Main navigation" onClick={() => setIsOpen(false)}>
        <a href="/" aria-current="page">Home</a>
        <a href="/login">Login</a>
        <a href="/register">Register</a>
      </nav>
    </header>
  );
}
