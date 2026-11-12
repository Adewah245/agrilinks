import {
  createURLPolicy,
  isBundledAsset,
  PREVIEW_API_PREFIX,
} from "./url-policy.js";

async function toBase64(blob) {
  const bytes = new Uint8Array(await blob.arrayBuffer());
  let binary = "";
  for (let offset = 0; offset < bytes.length; offset += 32768)
    binary += String.fromCharCode(...bytes.subarray(offset, offset + 32768));
  return btoa(binary);
}

const quotePart = (value) =>
  String(value).replace(
    /[\r\n"]/g,
    (character) => `%${character.charCodeAt(0).toString(16).toUpperCase()}`,
  );

// Send complete multipart bytes through the documented native file transport.
// This preserves repeated fields, UTF-8 text, and the original image/audio bytes
// on both platforms, including Android's native HTTP implementation.
export async function serializeBody(body, headers) {
  if (body == null) return {};
  if (body instanceof URLSearchParams) {
    if (!headers.has("Content-Type"))
      headers.set(
        "Content-Type",
        "application/x-www-form-urlencoded;charset=UTF-8",
      );
    return { data: body.toString() };
  }
  if (body instanceof FormData) {
    const boundary = `Agrilinks-${Array.from(
      crypto.getRandomValues(new Uint8Array(16)),
      (byte) => byte.toString(16).padStart(2, "0"),
    ).join("")}`;
    const parts = [];
    for (const [name, value] of body) {
      let disposition = `--${boundary}\r\nContent-Disposition: form-data; name="${quotePart(name)}"`;
      if (value instanceof Blob) {
        disposition += `; filename="${quotePart(value.name || "upload")}"\r\nContent-Type: ${value.type || "application/octet-stream"}`;
        parts.push(`${disposition}\r\n\r\n`, value, "\r\n");
      } else {
        parts.push(
          `${disposition}\r\n\r\n`,
          String(value).replace(/\r\n|\r|\n/g, "\r\n"),
          "\r\n",
        );
      }
    }
    parts.push(`--${boundary}--\r\n`);
    headers.set("Content-Type", `multipart/form-data; boundary=${boundary}`);
    return { data: await toBase64(new Blob(parts)), dataType: "file" };
  }
  if (
    body instanceof Blob ||
    body instanceof ArrayBuffer ||
    ArrayBuffer.isView(body)
  ) {
    const blob = body instanceof Blob ? body : new Blob([body]);
    if (!headers.has("Content-Type"))
      headers.set("Content-Type", blob.type || "application/octet-stream");
    if (
      /^(text\/|application\/(?:[^;]*\+)?json|application\/x-www-form-urlencoded)/i.test(
        headers.get("Content-Type") || blob.type,
      )
    )
      return { data: await blob.text() };
    return { data: await toBase64(blob), dataType: "file" };
  }
  if (!headers.has("Content-Type"))
    headers.set("Content-Type", "text/plain;charset=UTF-8");
  return { data: String(body) };
}

function abortable(promise, signal) {
  if (!signal) return promise;
  return new Promise((resolve, reject) => {
    const abort = () =>
      reject(
        signal.reason || new DOMException("Request cancelled", "AbortError"),
      );
    signal.addEventListener("abort", abort, { once: true });
    if (signal.aborted) abort();
    promise
      .then(resolve, reject)
      .finally(() => signal.removeEventListener("abort", abort));
  });
}

function responseFromNative(result, redirected, binary) {
  const headers = new Headers();
  for (const [key, value] of Object.entries(result.headers || {})) {
    if (!/^(set-cookie2?|content-length)$/i.test(key))
      headers.set(key, String(value));
  }
  let body =
    typeof result.data === "string"
      ? result.data
      : JSON.stringify(result.data ?? "");
  if (binary && result.status >= 200 && result.status < 300) {
    const bytes = Uint8Array.from(atob(body), (character) =>
      character.charCodeAt(0),
    );
    body = new Blob([bytes], {
      type: headers.get("Content-Type") || "application/octet-stream",
    });
  }
  const response = new Response(
    [204, 205, 304].includes(result.status) ? null : body,
    {
      status: result.status,
      headers,
    },
  );
  Object.defineProperties(response, {
    url: { value: result.url },
    redirected: { value: redirected },
  });
  return response;
}

export function createNativeFetch({
  origin,
  localURL,
  request,
  webFetch,
  platform,
}) {
  const policy = createURLPolicy(origin, localURL);
  return async function nativeFetch(input, init = {}) {
    const signal =
      init.signal === undefined
        ? input instanceof Request
          ? input.signal
          : undefined
        : init.signal;
    const destination = policy.resolve(
      input instanceof Request ? input.url : input,
      globalThis.location?.pathname || "/",
    );
    if (!destination.internal || isBundledAsset(destination.url.pathname))
      return webFetch(input, init);
    const headers = new Headers(
      init.headers || (input instanceof Request ? input.headers : undefined),
    );
    let method = (
      init.method || (input instanceof Request ? input.method : "GET")
    ).toUpperCase();
    let body = init.body;
    if (
      body === undefined &&
      input instanceof Request &&
      !["GET", "HEAD"].includes(method)
    )
      body = headers.get("Content-Type")?.startsWith("multipart/form-data")
        ? await input.clone().formData()
        : await input.clone().blob();
    if (signal?.aborted)
      throw (
        signal.reason || new DOMException("Request cancelled", "AbortError")
      );
    let encoded = await serializeBody(body, headers);
    headers.set("Origin", origin);
    const timeout =
      init.nativeTimeout || (["GET", "HEAD"].includes(method) ? 30000 : 120000);
    let url = destination.url;
    let redirected = false;
    for (let hop = 0; hop < 8; hop++) {
      if (signal?.aborted)
        throw (
          signal.reason || new DOMException("Request cancelled", "AbortError")
        );
      const requestURL = new URL(url);
      requestURL.hash = "";
      const result = await abortable(
        request({
          url: requestURL.href,
          method,
          headers: Object.fromEntries(headers),
          ...encoded,
          // iOS uses the connection timeout for the entire URLSession request.
          connectTimeout:
            platform === "ios" ? timeout : Math.min(15000, timeout),
          readTimeout: timeout,
          disableRedirects: true,
          responseType: init.nativeResponseType === "blob" ? "blob" : "text",
        }),
        signal,
      );
      const response = responseFromNative(
        { ...result, url: result.url || requestURL.href },
        redirected,
        init.nativeResponseType === "blob",
      );
      const location = response.headers.get("Location");
      if (
        ![301, 302, 303, 307, 308].includes(response.status) ||
        !location ||
        init.redirect === "manual"
      )
        return response;
      if (init.redirect === "error") throw new TypeError("Unexpected redirect");
      const next = policy.resolve(location, `${url.pathname}${url.search}`);
      // External checkout redirects are handled by the page navigator, which
      // opens the system browser without forwarding any account request body.
      if (!next.internal)
        throw new Error("This request redirected outside Agrilinks.");
      url = next.url;
      redirected = true;
      if (
        response.status === 303 ||
        ([301, 302].includes(response.status) && method === "POST")
      ) {
        method = "GET";
        encoded = {};
        headers.delete("Content-Type");
      }
    }
    throw new Error("Too many redirects. Please sign in again.");
  };
}

// A same-origin Vite proxy makes the native UI reviewable in a normal browser.
// Installed apps use native HTTP directly; no backend CORS exceptions are needed.
export function createPreviewFetch({ origin, localURL, webFetch }) {
  const policy = createURLPolicy(origin, localURL);
  return async (input, init = {}) => {
    const destination = policy.resolve(
      input instanceof Request ? input.url : input,
      globalThis.location?.pathname || "/",
    );
    if (!destination.internal || isBundledAsset(destination.url.pathname))
      return webFetch(input, init);
    const proxyURL = new URL(
      `${PREVIEW_API_PREFIX}${destination.url.pathname}${destination.url.search}`,
      localURL,
    );
    const response = await webFetch(
      input instanceof Request ? new Request(proxyURL, input) : proxyURL,
      {
        ...init,
        redirect: "follow",
        credentials: "same-origin",
      },
    );
    const responseURL = new URL(response.url || proxyURL);
    if (responseURL.pathname.startsWith(`${PREVIEW_API_PREFIX}/`)) {
      Object.defineProperty(response, "url", {
        value: new URL(
          `${responseURL.pathname.slice(PREVIEW_API_PREFIX.length)}${responseURL.search}${responseURL.hash}`,
          origin,
        ).href,
      });
    }
    return response;
  };
}
