import assert from "node:assert/strict";
import { test } from "node:test";
import {
  backendOrigin,
  createURLPolicy,
  PAGE_DATA_TYPE,
} from "../lib/url-policy.js";
import {
  createNativeFetch,
  createPreviewFetch,
  serializeBody,
} from "../lib/http.js";
import { createPageNavigator } from "../lib/navigation.js";
import { createMobileRuntime } from "../lib/runtime.js";

const origin = "https://agrilinks.example";
const localURL = "capacitor://localhost/";
const policy = createURLPolicy(origin, localURL);
const pageResponse = (page = "wallet.html", status = 200) =>
  new Response(
    JSON.stringify({
      page,
      title: "Wallet",
      styles: ["/static/css/wallet.css"],
      data: { User: { ID: 7 } },
    }),
    { status, headers: { "Content-Type": PAGE_DATA_TYPE } },
  );

test("release apps require an HTTPS origin and dev apps use each emulator's host", () => {
  assert.equal(
    backendOrigin("", { platform: "android" }),
    "http://10.0.2.2:9000",
  );
  assert.equal(backendOrigin("", { platform: "ios" }), "http://localhost:9000");
  assert.equal(backendOrigin(origin, { production: true }), origin);
  for (const value of [
    "",
    "http://example.com",
    "https://user:password@example.com",
    "https://example.com/api",
    "https://example.com?token=secret",
    "javascript:alert(1)",
  ])
    assert.throws(() => backendOrigin(value, { production: true }));
});

test("native routes work with iOS's custom origin and external URLs stay external", () => {
  assert.equal(
    policy.resolve("capacitor://localhost/wallet?notice=ok#funding").path,
    "/wallet?notice=ok#funding",
  );
  assert.equal(policy.resolve("/wallet").url.href, `${origin}/wallet`);
  assert.equal(policy.resolve("//other.example/wallet").internal, false);
  assert.equal(
    policy.resolve("https://agrilinks.example.other.example/wallet").internal,
    false,
  );
  assert.equal(
    policy.deepLink("agrilinks://wallet#funding").path,
    "/wallet#funding",
  );
  assert.equal(policy.deepLink("agrilinks:///wallet").path, "/wallet");
  assert.throws(() => policy.deepLink("https://other.example/wallet"));
});

test("multipart uploads preserve Unicode, repeated fields, filenames and binary bytes", async () => {
  const form = new FormData();
  form.append("name", "Adé Ọkafor");
  form.append("tags", "one");
  form.append("tags", "two");
  form.append("note", "first\nsecond");
  const image = new Uint8Array([0, 255, 128, 42]);
  form.append(
    "selfie",
    new Blob([image], { type: "image/jpeg" }),
    'selfie"name.jpg',
  );
  const headers = new Headers();
  const body = await serializeBody(form, headers);
  assert.equal(body.dataType, "file");
  const decoded = Buffer.from(body.data, "base64");
  const text = decoded.toString("utf8");
  const boundary = headers.get("Content-Type").split("boundary=")[1];
  assert.ok(text.includes("Adé Ọkafor"));
  assert.equal(text.match(/name="tags"/g).length, 2);
  assert.ok(text.includes("first\r\nsecond"));
  assert.ok(text.includes('filename="selfie%22name.jpg"'));
  assert.ok(decoded.includes(Buffer.from(image)));
  assert.ok(text.endsWith(`--${boundary}--\r\n`));
});

test("JSON Request and Blob bodies reach the backend as UTF-8 JSON", async () => {
  const requests = [];
  const fetch = createNativeFetch({
    origin,
    localURL,
    platform: "android",
    webFetch: () => assert.fail("unexpected web request"),
    request: async (options) => {
      requests.push(options);
      return {
        status: 200,
        headers: { "Content-Type": "application/json" },
        data: { ok: true },
        url: options.url,
      };
    },
  });
  const json = JSON.stringify({ name: "Adé Ọkafor" });
  await fetch(
    new Request(`${origin}/example`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: json,
    }),
  );
  await fetch("/example", {
    method: "POST",
    body: new Blob([json], { type: "application/json" }),
  });
  for (const options of requests) {
    assert.equal(options.data, json);
    assert.equal(options.dataType, undefined);
    assert.equal(options.headers["content-type"], "application/json");
  }
});

test("a cancelled Request cannot send a mutation through native HTTP", async () => {
  const cancelled = new AbortController();
  cancelled.abort();
  const fetch = createNativeFetch({
    origin,
    localURL,
    platform: "ios",
    webFetch: () => assert.fail("unexpected web request"),
    request: () => assert.fail("cancelled request reached the network"),
  });
  await assert.rejects(
    fetch(
      new Request(`${origin}/wallet`, {
        method: "POST",
        body: "action=deposit",
        signal: cancelled.signal,
      }),
    ),
    { name: "AbortError" },
  );
});

test("browser preview preserves a Request's POST body and headers", async () => {
  const fetch = createPreviewFetch({
    origin,
    localURL: "http://localhost:5174/",
    webFetch: async (request, options) => {
      assert.equal(request.url, "http://localhost:5174/__native-api/wallet");
      assert.equal(request.method, "POST");
      assert.equal(request.headers.get("Content-Type"), "application/json");
      assert.equal(await request.text(), '{"name":"Adé"}');
      assert.equal(options.credentials, "same-origin");
      return new Response("ok");
    },
  });
  await fetch(
    new Request(`${origin}/wallet`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: '{"name":"Adé"}',
    }),
  );
});

test("a hash deep link cancels an older page load and keeps its screen", async (t) => {
  const savedWindow = globalThis.window;
  const savedDocument = globalThis.document;
  t.after(() => {
    globalThis.window = savedWindow;
    globalThis.document = savedDocument;
  });
  globalThis.window = {
    location: { pathname: "/wallet", search: "", hash: "" },
    history: { state: {}, replaceState() {}, pushState() {} },
    scrollTo() {},
  };
  globalThis.document = { title: "" };
  let release;
  let previousSignal;
  const runtime = createMobileRuntime({
    policy,
    openURL: () => assert.fail("unexpected external link"),
    hasPage: () => true,
    loadStyles: async () => {},
    fetch: async (url, options) => {
      if (url.endsWith("/marketplace")) {
        previousSignal = options.signal;
        await new Promise((resolve) => {
          release = resolve;
        });
        return pageResponse("marketplace.html");
      }
      return pageResponse();
    },
  });
  await runtime.navigate("/wallet", { replace: true });
  const older = runtime.navigate("/marketplace");
  assert.equal(runtime.getState().loading, true);
  await runtime.navigate("/wallet#funding");
  assert.equal(previousSignal.aborted, true);
  assert.equal(runtime.getState().loading, false);
  release();
  await older;
  assert.equal(runtime.getState().path, "/wallet#funding");
  assert.equal(runtime.getState().page.page, "wallet.html");
});

test("login redirects use native cookies and turn the following request into GET", async () => {
  const requests = [];
  const fetch = createNativeFetch({
    origin,
    localURL,
    platform: "android",
    webFetch: () => assert.fail("unexpected web request"),
    request: async (options) => {
      requests.push(options);
      if (requests.length === 1)
        return {
          status: 303,
          headers: {
            Location: "/dashboard",
            "Set-Cookie": "session=private; HttpOnly",
          },
          data: "",
          url: `${origin}/login`,
        };
      return {
        status: 200,
        headers: { "Content-Type": PAGE_DATA_TYPE },
        data: { page: "dashboard.html" },
        url: `${origin}/dashboard`,
      };
    },
  });
  const response = await fetch("/login", {
    method: "POST",
    body: new URLSearchParams({
      phone: "08000000000",
      password: "sample-password",
    }),
  });
  assert.equal(requests[0].headers.origin, origin);
  assert.match(
    requests[0].headers["content-type"],
    /application\/x-www-form-urlencoded/,
  );
  assert.equal(requests[0].disableRedirects, true);
  assert.equal(requests[1].method, "GET");
  assert.equal(requests[1].data, undefined);
  assert.equal(response.url, `${origin}/dashboard`);
  assert.equal(response.redirected, true);
  assert.equal(response.headers.has("Set-Cookie"), false);
  assert.equal((await response.json()).page, "dashboard.html");
});

test("an external checkout does not receive a replay of account credentials or the form", async () => {
  const requests = [];
  const opened = [];
  const fetch = createNativeFetch({
    origin,
    localURL,
    platform: "ios",
    webFetch: () => assert.fail("unexpected web request"),
    request: async (options) => {
      requests.push(options);
      return {
        status: 303,
        headers: { Location: "https://checkout.example/pay" },
        data: "",
        url: `${origin}/wallet`,
      };
    },
  });
  const load = createPageNavigator({
    fetch,
    policy,
    openURL: async (url) => opened.push(url),
    hasPage: () => true,
  });
  assert.deepEqual(
    await load("/wallet", {
      method: "POST",
      body: new URLSearchParams({ pin: "123456", action: "deposit" }),
    }),
    { external: true },
  );
  assert.equal(requests.length, 1);
  assert.equal(requests[0].url, `${origin}/wallet`);
  assert.equal(requests[0].connectTimeout, requests[0].readTimeout);
  assert.deepEqual(opened, ["https://checkout.example/pay"]);
});

test("failed mutations are sent once and aborting a poll does not cause retries", async () => {
  let count = 0;
  const fetch = createNativeFetch({
    origin,
    localURL,
    platform: "android",
    webFetch: () => assert.fail("unexpected web request"),
    request: async () => {
      count++;
      throw new Error("connection lost");
    },
  });
  await assert.rejects(
    fetch("/wallet", {
      method: "POST",
      body: new URLSearchParams({ action: "withdraw" }),
    }),
    /connection lost/,
  );
  assert.equal(count, 1);
  const cancelled = new AbortController();
  cancelled.abort();
  await assert.rejects(fetch("/wallet/status", { signal: cancelled.signal }), {
    name: "AbortError",
  });
  assert.equal(count, 1);
});

test("page redirects preserve query strings and show server validation errors", async () => {
  const requests = [];
  const load = createPageNavigator({
    policy,
    openURL: () => assert.fail("unexpected external link"),
    hasPage: (name) => name === "wallet.html",
    fetch: async (url, options) => {
      requests.push({ url, options });
      if (requests.length === 1)
        return new Response(null, {
          status: 303,
          headers: { Location: "/wallet?notice=check#funding" },
        });
      return pageResponse("wallet.html", 422);
    },
  });
  const result = await load("/wallet", {
    method: "POST",
    body: new URLSearchParams({ action: "setup_funding" }),
  });
  assert.equal(requests[0].options.headers.get("Accept"), PAGE_DATA_TYPE);
  assert.equal(requests[1].options.method, "GET");
  assert.equal(requests[1].options.body, undefined);
  assert.equal(result.path, "/wallet?notice=check#funding");
  assert.equal(result.page.page, "wallet.html");
});

test("the client rejects HTML and unknown screens instead of executing remote code", async () => {
  for (const response of [
    new Response("<script>alert(1)</script>", {
      headers: { "Content-Type": "text/html" },
    }),
    pageResponse("unknown.html"),
  ]) {
    const load = createPageNavigator({
      policy,
      openURL: () => assert.fail("unexpected external link"),
      hasPage: (name) => name === "wallet.html",
      fetch: async () => response,
    });
    await assert.rejects(load("/wallet"), /could not be loaded/);
  }
});

test("bundled assets use the WebView while private images use authenticated native HTTP", async () => {
  const nativeRequests = [];
  const localRequests = [];
  const fetch = createNativeFetch({
    origin,
    localURL,
    platform: "android",
    webFetch: async (url) => {
      localRequests.push(url);
      return new Response("logo");
    },
    request: async (options) => {
      nativeRequests.push(options);
      return {
        status: 200,
        headers: { "Content-Type": "image/jpeg" },
        data: Buffer.from([1, 2, 3]).toString("base64"),
        url: options.url,
      };
    },
  });
  await fetch("/static/assets/logo/agrilinks-nigeria-logo.svg");
  const response = await fetch("/private/verification/7", {
    nativeResponseType: "blob",
  });
  assert.deepEqual(localRequests, [
    "/static/assets/logo/agrilinks-nigeria-logo.svg",
  ]);
  assert.equal(nativeRequests[0].url, `${origin}/private/verification/7`);
  assert.equal(nativeRequests[0].responseType, "blob");
  assert.deepEqual(
    [...new Uint8Array(await response.arrayBuffer())],
    [1, 2, 3],
  );
});
