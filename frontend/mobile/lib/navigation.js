import { PAGE_DATA_TYPE } from "./url-policy.js";

export function createPageNavigator({ fetch, policy, openURL, hasPage }) {
  return async function loadPage(path, options = {}) {
    let destination = policy.resolve(path, options.currentPath || "/");
    if (!destination.internal) {
      await openURL(destination.url.href);
      return { external: true };
    }
    const headers = new Headers(options.headers);
    headers.set("Accept", PAGE_DATA_TYPE);
    let method = (options.method || "GET").toUpperCase();
    let body = options.body;
    for (let hop = 0; hop < 8; hop++) {
      const response = await fetch(destination.url.href, {
        method,
        body,
        headers,
        redirect: "manual",
        signal: options.signal,
      });
      const previewExternal = response.headers.get("X-Agrilinks-External-URL");
      const redirect =
        [301, 302, 303, 307, 308].includes(response.status) &&
        response.headers.get("Location");
      if (redirect || previewExternal) {
        const next = policy.resolve(
          redirect || previewExternal,
          destination.path,
        );
        if (!next.internal) {
          await openURL(next.url.href);
          return { external: true };
        }
        destination = next;
        if (
          response.status === 303 ||
          ([301, 302].includes(response.status) && method === "POST")
        ) {
          method = "GET";
          body = undefined;
          headers.delete("Content-Type");
        }
        continue;
      }
      if (!response.headers.get("Content-Type")?.startsWith(PAGE_DATA_TYPE))
        throw new Error("This page could not be loaded. Please try again.");
      const page = await response.json();
      if (
        typeof page.page !== "string" ||
        !hasPage(page.page) ||
        !Array.isArray(page.styles) ||
        typeof page.title !== "string"
      )
        throw new Error("This page could not be loaded. Please try again.");
      if (response.redirected) {
        destination = policy.resolve(response.url, destination.path);
        if (!destination.internal)
          throw new Error("Unexpected page destination.");
      }
      return { page, path: destination.path };
    }
    throw new Error("This page could not be loaded. Please sign in again.");
  };
}
