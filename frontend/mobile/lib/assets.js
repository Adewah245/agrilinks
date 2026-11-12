import { isServerAsset } from "./url-policy.js";

export function createStyleLoader() {
  return async (styles, isCurrent) => {
    const links = styles
      .filter(
        (style) =>
          style.startsWith("/static/css/") ||
          style.includes("cdnjs.cloudflare.com/ajax/libs/font-awesome/"),
      )
      .map((style) => {
        const link = document.createElement("link");
        link.rel = "stylesheet";
        link.dataset.mobileStyle = "pending";
        link.href = style.startsWith("/static/")
          ? style
          : "/static/vendor/fontawesome/css/all.min.css";
        return link;
      });
    try {
      await Promise.all(
        links.map(
          (link) =>
            new Promise((resolve, reject) => {
              link.onload = resolve;
              link.onerror = () =>
                reject(new Error("A page style could not be loaded."));
              document.head.append(link);
            }),
        ),
      );
      if (!isCurrent()) {
        links.forEach((link) => link.remove());
        return;
      }
      document
        .querySelectorAll('link[data-mobile-style="active"]')
        .forEach((link) => link.remove());
      links.forEach((link) => {
        link.dataset.mobileStyle = "active";
      });
    } catch (error) {
      links.forEach((link) => link.remove());
      throw error;
    }
  };
}

export function prepareUploadedMedia(root, runtime) {
  const controller = new AbortController();
  const blobs = new Set();
  const requested = new WeakMap();
  const update = async (element, attribute) => {
    const source = element.getAttribute(attribute);
    if (!source || requested.get(element)?.has(source)) return;
    const destination = runtime.policy.resolve(source, runtime.getState().path);
    if (!destination.internal || !isServerAsset(destination.url.pathname))
      return;
    if (!requested.has(element)) requested.set(element, new Set());
    requested.get(element).add(source);
    try {
      // Native HTTP sends the appropriate session cookie for private pictures.
      // Blob URLs stay in memory and are revoked when this page is unmounted.
      const response = await runtime.fetch(destination.url.href, {
        signal: controller.signal,
        nativeResponseType: "blob",
      });
      if (
        !response.ok ||
        !response.headers.get("Content-Type")?.startsWith("image/")
      )
        return;
      const blob = await response.blob();
      if (controller.signal.aborted || !element.isConnected) return;
      const url = URL.createObjectURL(blob);
      blobs.add(url);
      element.setAttribute(attribute, url);
    } catch {
      /* An unavailable photo must not interrupt the rest of a page. */
    }
  };
  const scan = () =>
    root
      .querySelectorAll("img[src],video[poster]")
      .forEach((element) =>
        update(element, element.tagName === "IMG" ? "src" : "poster"),
      );
  const observer = new MutationObserver(scan);
  observer.observe(root, {
    subtree: true,
    childList: true,
    attributes: true,
    attributeFilter: ["src", "poster"],
  });
  scan();
  return () => {
    controller.abort();
    observer.disconnect();
    blobs.forEach((url) => URL.revokeObjectURL(url));
  };
}
