export default function Footer() {
  return (
    <footer className="footer">
      <div className="footer-container">
        <div className="footer-brand">
          <img
            className="footer-brand-logo"
            src="/static/assets/logo/agrilinks-nigeria-logo.svg"
            alt="Agrilinks Nigeria logo"
            loading="lazy"
            width="160"
            height="45"
          />
          <p>
            Empowering farmers through technology, AI-powered agricultural assistance,
            farm management tools, and digital agricultural commerce.
          </p>
        </div>

        <nav className="footer-links" aria-label="Footer navigation">
          <h2>Quick Links</h2>
          <ul>
            <li><a href="/">Home</a></li>
            <li><a href="/learning">Learn with AI</a></li>
            <li><a href="/login">Login</a></li>
            <li><a href="/register">Register</a></li>
          </ul>
        </nav>

        <div className="footer-contact">
          <h2>Contact</h2>
          <p>Email: <a href="mailto:support@agroshield.com">Email support</a></p>
          <p>Phone: <a href="tel:+2349060904492">+234 9060904492</a></p>
        </div>
      </div>

      <hr />
      <p className="copyright">&copy; 2026 Agrilinks Nigeria. All rights reserved.</p>
    </footer>
  );
}
