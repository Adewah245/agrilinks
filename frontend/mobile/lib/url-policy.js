export const PAGE_DATA_TYPE = "application/vnd.agrilinks.page.v1+json";
export const PREVIEW_API_PREFIX = "/__native-api";

export function backendOrigin(
  value,
  { production = false, platform = "web" } = {},
) {
  if (!value && production)
    throw new Error(
      "Set VITE_MOBILE_BACKEND_URL to your HTTPS Go server before building a release app.",
    );
  const url = new URL(
    value ||
      (platform === "android"
        ? "http://10.0.2.2:9000"
        : "http://localhost:9000"),
  );
  if (
    !["http:", "https:"].includes(url.protocol) ||
    url.username ||
    url.password ||
    url.pathname !== "/" ||
    url.search ||
    url.hash
  )
    throw new Error(
      "VITE_MOBILE_BACKEND_URL must be an HTTP(S) origin without credentials, a path, or query parameters.",
    );
  if (production && url.protocol !== "https:")
    throw new Error("Release mobile apps require an HTTPS backend.");
  return url.origin;
}

export function createURLPolicy(origin, localOrigin) {
  const backend = new URL(origin);
  const local = new URL(localOrigin);
  const isInternal = (url) =>
    url.origin === backend.origin ||
    (url.protocol === local.protocol && url.host === local.host);
  function resolve(value, currentPath = "/") {
    const url = new URL(String(value), new URL(currentPath, backend));
    if (!isInternal(url)) return { internal: false, url };
    const path = `${url.pathname}${url.search}${url.hash}`;
    return { internal: true, path, url: new URL(path, backend) };
  }
  function deepLink(value) {
    const url = new URL(value);
    if (url.protocol === "agrilinks:")
      return resolve(
        `${url.host ? `/${url.host}${url.pathname}` : url.pathname || "/"}${url.search}${url.hash}`,
      );
    const destination = resolve(value);
    if (!destination.internal)
      throw new Error("This link belongs to another app.");
    return destination;
  }
  return { resolve, deepLink, isInternal, origin: backend.origin };
}

export function isBundledAsset(path) {
  return (
    path.startsWith("/static/assets/") ||
    path.startsWith("/static/css/") ||
    path.startsWith("/static/vendor/")
  );
}

export function isServerAsset(path) {
  return (
    path.startsWith("/static/uploads/") ||
    path.startsWith("/uploads/") ||
    path.startsWith("/private/verification/")
  );
}
