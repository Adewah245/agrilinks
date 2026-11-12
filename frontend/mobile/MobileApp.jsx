import {
  Component,
  lazy,
  Suspense,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useSyncExternalStore,
} from "react";
import { pages } from "../src/pages/registry.js";
import { prepareUploadedMedia } from "./lib/assets.js";

class ScreenBoundary extends Component {
  state = { failed: false };
  static getDerivedStateFromError() {
    return { failed: true };
  }
  render() {
    if (this.state.failed)
      return (
        <main className="mobile-connection" role="alert">
          <h1>This screen couldn’t be loaded.</h1>
          <p>Please try again.</p>
          <button type="button" onClick={this.props.retry}>
            Try again
          </button>
        </main>
      );
    return this.props.children;
  }
}

const tabs = [
  {
    path: "/dashboard",
    label: "Home",
    shape: "m3 10 9-7 9 7v10H3Zm6 10v-7h6v7",
  },
  {
    path: "/marketplace",
    label: "Shop",
    shape: "M5 8h14l2 12H3Zm3 0V6a4 4 0 0 1 8 0v2",
  },
  {
    path: "/learning",
    label: "Learn",
    shape:
      "M12 21v-9m0 3C4 15 3 6 3 6c9 0 9 9 9 9Zm0-4c0-8 9-9 9-9 0 8-9 9-9 9Z",
  },
  {
    path: "/wallet",
    label: "Wallet",
    shape: "M3 7V5h16v4M3 7h18v13H3Zm12 6h6v4h-6Z",
  },
  {
    path: "/profile",
    label: "Account",
    shape: "M16 7a4 4 0 1 1-8 0 4 4 0 0 1 8 0ZM4 21v-2a8 8 0 0 1 16 0v2",
  },
];

function NativeScreen({ state, runtime }) {
  const Page = useMemo(() => lazy(pages[state.page.page]), [state.page.page]);
  const root = useRef(null);
  useEffect(
    () => prepareUploadedMedia(root.current, runtime),
    [runtime, state.revision],
  );
  return (
    <div
      ref={root}
      className="mobile-screen"
      inert={state.loading || undefined}
    >
      <ScreenBoundary key={state.revision} retry={runtime.refresh}>
        <Suspense
          fallback={
            <p className="mobile-loading" role="status">
              Loading your screen…
            </p>
          }
        >
          <Page key={state.revision} data={state.page.data || {}} />
        </Suspense>
      </ScreenBoundary>
    </div>
  );
}

export default function MobileApp({ runtime }) {
  const state = useSyncExternalStore(runtime.subscribe, runtime.getState);
  const signedIn = Boolean(state.page?.data?.User);
  useLayoutEffect(() => {
    document.body.className = `agrilinks-native ${state.page?.bodyClass || ""}${signedIn ? " has-mobile-tabs" : ""}`;
  }, [state.page?.bodyClass, signedIn]);
  if (!state.page)
    return (
      <main
        className="mobile-connection"
        role={state.error ? "alert" : "status"}
      >
        <img
          src="/static/assets/logo/agrilinks-nigeria-logo.svg"
          width="240"
          height="64"
          alt="Agrilinks Nigeria"
        />
        {state.loading ? (
          <>
            <span className="mobile-spinner" aria-hidden="true" />
            <h1>Welcome to Agrilinks</h1>
            <p>Connecting to your agricultural community…</p>
          </>
        ) : (
          <>
            <h1>Let’s reconnect</h1>
            <p>{state.error}</p>
            <button type="button" onClick={runtime.refresh}>
              Try again
            </button>
          </>
        )}
      </main>
    );
  return (
    <>
      {state.loading && (
        <div className="mobile-progress" role="status">
          <span className="mobile-spinner" aria-hidden="true" />
          {state.busy ? "Saving your change…" : "Loading…"}
        </div>
      )}
      {state.error && (
        <aside className="mobile-notice" role="alert">
          <p>{state.error}</p>
          <button
            type="button"
            onClick={runtime.refresh}
            disabled={state.loading}
          >
            Refresh page
          </button>
        </aside>
      )}
      {state.notice && (
        <aside className="mobile-notice" role="status">
          <p>{state.notice}</p>
          <button type="button" onClick={() => runtime.showNotice("")}>
            Dismiss
          </button>
        </aside>
      )}
      <NativeScreen state={state} runtime={runtime} />
      {signedIn && (
        <nav
          className="mobile-tabs"
          aria-label="App navigation"
          inert={state.loading || undefined}
        >
          {tabs.map((tab) => (
            <a
              key={tab.path}
              href={tab.path}
              aria-current={
                state.path.split(/[?#]/)[0] === tab.path ? "page" : undefined
              }
            >
              <svg
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="1.7"
                strokeLinecap="round"
                strokeLinejoin="round"
                aria-hidden="true"
              >
                <path d={tab.shape} />
              </svg>
              <span>{tab.label}</span>
            </a>
          ))}
        </nav>
      )}
    </>
  );
}
