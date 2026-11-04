import { Component, lazy, Suspense } from "react";
import { createRoot } from "react-dom/client";
import { pages } from "./pages/registry.js";
import "./styles/index.css";

class PageErrorBoundary extends Component {
  state = { failed: false };
  static getDerivedStateFromError() {
    return { failed: true };
  }
  render() {
    if (this.state.failed)
      return (
        <main className="page-load-message" role="alert">
          <h1>This page could not be loaded.</h1>
          <p>Please refresh the page and try again.</p>
          <button type="button" onClick={() => window.location.reload()}>
            Try again
          </button>
        </main>
      );
    return this.props.children;
  }
}

const bootstrap = JSON.parse(document.getElementById("page-data").textContent);
const Page = lazy(
  pages[bootstrap.page] || (() => Promise.reject(new Error("Unknown page"))),
);

createRoot(document.getElementById("root")).render(
  <PageErrorBoundary>
    <Suspense
      fallback={
        <p className="page-load-message" role="status">
          Loading…
        </p>
      }
    >
      <Page data={bootstrap.data || {}} />
    </Suspense>
  </PageErrorBoundary>,
);
