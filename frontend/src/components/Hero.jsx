import HeroSlideshow from './HeroSlideshow.jsx';

export default function Hero() {
  return (
    <section className="hero" aria-labelledby="hero-title">
      <div className="hero-content">
        <h1 id="hero-title">
          Agrilinks Nigeria is a smart operating platform for every farmer.{' '}
          <span className="hero-tagline">
            Helping protect your farm, reduce losses, and connect you to better opportunities.
          </span>
        </h1>

        <p className="hero-subtitle">
          From crops and livestock to poultry, fisheries, beekeeping, and other agricultural
          activities, Agrilinks Nigeria helps you manage your farm and connect with trusted markets.
        </p>

        <p className="hero-description">
          Manage agricultural products, monitor your farm, connect with buyers, receive AI-powered
          agricultural assistance, access market opportunities, and grow your agricultural business.
        </p>

        <div className="hero-buttons">
          <a href="/register" className="primary-btn">Get Started</a>
          <a href="/login" className="secondary-btn">Login</a>
          <a href="/learning" className="secondary-btn">Learn farming — no account needed</a>
        </div>
      </div>

      <HeroSlideshow />
    </section>
  );
}
