import assert from "node:assert/strict";
import { test } from "node:test";
import { fetchWithPageSignal } from "../src/lib/page-fetch.js";

test("leaving a page cancels its requests even when they have a caller signal", async () => {
  for (const source of ["page", "caller"]) {
    const page = new AbortController();
    const caller = new AbortController();
    const request = fetchWithPageSignal(
      (_, { signal }) =>
        new Promise((resolve, reject) =>
          signal.addEventListener("abort", () => reject(signal.reason), {
            once: true,
          }),
        ),
      "/wallet/status",
      { signal: caller.signal },
      page.signal,
    );
    const reason = new DOMException(`Cancelled by ${source}`, "AbortError");
    (source === "page" ? page : caller).abort(reason);
    await assert.rejects(request, (error) => error === reason);
  }
});

test("an already-cancelled request stays cancelled and preserves its reason", async () => {
  const page = new AbortController();
  const caller = new AbortController();
  const reason = new DOMException("Screen closed", "AbortError");
  caller.abort(reason);
  await assert.rejects(
    fetchWithPageSignal(
      (_, { signal }) => {
        assert.equal(signal.aborted, true);
        return Promise.reject(signal.reason);
      },
      "/wallet/status",
      { signal: caller.signal },
      page.signal,
    ),
    (error) => error === reason,
  );
});

test("completed and failed requests release cancellation listeners", async () => {
  for (const fails of [false, true]) {
    const page = new AbortController();
    const caller = new AbortController();
    const removed = [];
    for (const signal of [page.signal, caller.signal]) {
      const remove = signal.removeEventListener.bind(signal);
      signal.removeEventListener = (...args) => {
        removed.push(signal);
        remove(...args);
      };
    }
    const request = fetchWithPageSignal(
      async () => {
        if (fails) throw new Error("Connection lost");
        return new Response("ready");
      },
      "/wallet/status",
      { signal: caller.signal },
      page.signal,
    );
    if (fails) await assert.rejects(request, /Connection lost/);
    else assert.equal(await (await request).text(), "ready");
    assert.deepEqual(removed, [page.signal, caller.signal]);
  }
});
