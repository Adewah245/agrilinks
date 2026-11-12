import { createPageNavigator } from "./navigation.js";

export function createMobileRuntime({
  fetch,
  policy,
  openURL,
  hasPage,
  loadStyles,
  captureSelfie,
  locate,
}) {
  const listeners = new Set();
  let state = {
    page: null,
    path: "/dashboard",
    loading: true,
    busy: false,
    error: "",
    notice: "",
    revision: 0,
  };
  let generation = 0;
  let dirty = false;
  let controller;
  const loadPage = createPageNavigator({ fetch, policy, openURL, hasPage });
  const update = (values) => {
    state = { ...state, ...values };
    listeners.forEach((listener) => listener());
  };

  async function navigate(
    path,
    { replace = false, pop = false, method = "GET", body } = {},
  ) {
    if (state.busy) return false;
    const destination = policy.resolve(path, state.path);
    if (!destination.internal) {
      try {
        await openURL(destination.url.href);
      } catch {
        update({ error: "That link could not be opened. Please try again." });
      }
      return false;
    }
    controller?.abort();
    const request = ++generation;
    if (
      state.page &&
      method === "GET" &&
      !replace &&
      destination.path !== state.path &&
      destination.url.pathname + destination.url.search ===
        new URL(state.path, policy.origin).pathname +
          new URL(state.path, policy.origin).search
    ) {
      if (!pop) window.location.hash = destination.url.hash;
      update({ path: destination.path, loading: false, error: "" });
      return true;
    }
    controller = new AbortController();
    const mutation = method !== "GET";
    update({ loading: true, busy: mutation, error: "" });
    try {
      const result = await loadPage(destination.path, {
        method,
        body,
        currentPath: state.path,
        // A mutation is sent once and allowed to finish. Route changes never
        // retry it or turn a transport failure into a second money action.
        signal: mutation ? undefined : controller.signal,
      });
      if (request !== generation) return false;
      if (result.external) {
        update({ loading: false, busy: false });
        return false;
      }
      await loadStyles(result.page.styles, () => request === generation);
      if (request !== generation) return false;
      if (!pop) {
        const index = Number(window.history.state?.agrilinksIndex || 0);
        const current = `${window.location.pathname}${window.location.search}${window.location.hash}`;
        const shouldReplace = replace || mutation || current === result.path;
        window.history[shouldReplace ? "replaceState" : "pushState"](
          { agrilinksIndex: shouldReplace ? index : index + 1 },
          "",
          result.path,
        );
      }
      document.title = result.page.title;
      dirty = false;
      update({
        page: result.page,
        path: result.path,
        loading: false,
        busy: false,
        error: "",
        revision: state.revision + 1,
      });
      window.scrollTo({ top: 0, behavior: "instant" });
      return true;
    } catch (error) {
      if (request !== generation || error.name === "AbortError") return false;
      update({
        loading: false,
        busy: false,
        error: mutation
          ? "We couldn’t confirm that change. Refresh the page to check its status before trying again."
          : "Couldn’t connect to Agrilinks. Check your internet connection and try again.",
      });
      // Restore the existing form guards without re-submitting any form.
      window.dispatchEvent(new Event("pageshow"));
      return false;
    }
  }

  const runtime = {
    fetch,
    policy,
    captureSelfie,
    locate,
    navigate,
    refresh: () =>
      state.loading || state.busy
        ? Promise.resolve(false)
        : navigate(state.path, { replace: true }),
    cancelNavigation: () => {
      if (state.busy) return;
      controller?.abort();
      generation++;
      update({ loading: false });
    },
    showNotice: (notice) => update({ notice }),
    refreshWhenSafe: () =>
      !dirty && !state.loading && !document.activeElement?.closest("form")
        ? navigate(state.path, { replace: true })
        : Promise.resolve(false),
    getState: () => state,
    subscribe: (listener) => {
      listeners.add(listener);
      return () => listeners.delete(listener);
    },
    showConnection: (connected) => {
      if (!connected)
        update({
          error: "You’re offline. Connect to the internet to continue.",
        });
      else if (
        state.error === "You’re offline. Connect to the internet to continue."
      )
        update({ error: "" });
    },
    install() {
      const previousFetch = window.fetch;
      window.fetch = fetch;
      window.agrilinksApp = runtime;
      document.documentElement.classList.add("agrilinks-native");
      const snapshots = new WeakMap();
      const capture = (event) => {
        const form = event.target;
        if (form instanceof HTMLFormElement) {
          // The submitter overload of FormData arrived after iOS 16.0 and
          // Chrome 107. Capture its value before form guards disable buttons.
          const body = new FormData(form);
          if (event.submitter?.name && !event.submitter.matches(":disabled"))
            body.append(event.submitter.name, event.submitter.value);
          snapshots.set(event, {
            body,
            action: event.submitter?.getAttribute("formaction") || form.action,
            method: event.submitter?.getAttribute("formmethod") || form.method,
            enctype: form.enctype,
          });
        }
      };
      const submit = (event) => {
        if (
          event.defaultPrevented ||
          !(event.target instanceof HTMLFormElement)
        )
          return;
        const snapshot = snapshots.get(event);
        const destination = policy.resolve(snapshot.action, state.path);
        if (!destination.internal) return;
        event.preventDefault();
        if (state.loading || state.busy) return;
        const fields = new URLSearchParams(
          [...snapshot.body].map(([key, value]) => [
            key,
            value instanceof File ? value.name : value,
          ]),
        );
        if (snapshot.method.toUpperCase() === "GET") {
          destination.url.search = fields.toString();
          navigate(
            `${destination.url.pathname}${destination.url.search}${destination.url.hash}`,
          );
        } else {
          navigate(destination.path, {
            method: snapshot.method.toUpperCase(),
            body:
              snapshot.enctype === "multipart/form-data"
                ? snapshot.body
                : fields,
          });
        }
      };
      const click = (event) => {
        const link = event.target.closest?.("a[href]");
        if (
          !link ||
          event.defaultPrevented ||
          event.button !== 0 ||
          event.metaKey ||
          event.ctrlKey ||
          event.altKey ||
          event.shiftKey ||
          link.hasAttribute("download")
        )
          return;
        const destination = policy.resolve(link.href, state.path);
        event.preventDefault();
        if (state.loading || state.busy) return;
        if (link.target === "_blank") {
          openURL(destination.url.href).catch(() =>
            update({ error: "That link could not be opened." }),
          );
          return;
        }
        navigate(
          destination.internal ? destination.path : destination.url.href,
        );
      };
      const markDirty = (event) => {
        if (event.target.closest?.("form")) dirty = true;
      };
      const pop = () =>
        navigate(
          `${window.location.pathname}${window.location.search}${window.location.hash}`,
          { pop: true },
        );
      document.addEventListener("submit", capture, true);
      document.addEventListener("submit", submit);
      document.addEventListener("click", click);
      document.addEventListener("input", markDirty);
      document.addEventListener("change", markDirty);
      window.addEventListener("popstate", pop);
      return () => {
        controller?.abort();
        generation++;
        document.removeEventListener("submit", capture, true);
        document.removeEventListener("submit", submit);
        document.removeEventListener("click", click);
        document.removeEventListener("input", markDirty);
        document.removeEventListener("change", markDirty);
        window.removeEventListener("popstate", pop);
        if (window.agrilinksApp === runtime) {
          delete window.agrilinksApp;
          window.fetch = previousFetch;
        }
      };
    },
  };
  return runtime;
}
