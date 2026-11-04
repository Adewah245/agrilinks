import assert from "node:assert/strict";
import { test } from "node:test";
import { readFile } from "node:fs/promises";
import {
  formatDate,
  humanOrderStatus,
  humanPaymentStatus,
  money,
  money0,
  printf,
  truth,
  withValue,
} from "../src/lib/view.js";

test("empty collections and optional objects use the correct empty states", () => {
  assert.equal(truth([]), false);
  assert.equal(truth(null), false);
  assert.equal(truth([{}]), true);
  assert.equal(
    withValue(
      null,
      () => "present",
      () => "empty",
    ),
    "empty",
  );
});

test("existing prices and status labels retain their presentation", () => {
  assert.equal(money(1250.5), "1,250.50");
  assert.equal(money0(1250), "1,250");
  assert.equal(printf("%+.0f%%", -12), "-12%");
  assert.equal(printf("%+.0f%%", 12), "+12%");
  assert.equal(humanOrderStatus("pending"), "Waiting for payment");
  assert.equal(humanPaymentStatus("successful"), "Paid");
});

test("UTC timestamps preserve dates, midnight, and noon", () => {
  assert.equal(
    formatDate("2026-11-04T00:05:09Z", "02 Jan 2006 03:04 PM"),
    "04 Nov 2026 12:05 AM",
  );
  assert.equal(
    formatDate("2026-11-04T12:05:09Z", "2006-01-02T15:04:05Z07:00"),
    "2026-11-04T12:05:09Z",
  );
  assert.equal(formatDate(null, "02 Jan 2006"), "");
});

test("all Go pages have a React entry and page-specific style assets", async () => {
  const manifest = JSON.parse(
    await readFile(
      new URL("../../backend/internal/render/pages.json", import.meta.url),
    ),
  );
  const { pages } = await import("../src/pages/registry.js");
  assert.equal(Object.keys(manifest).length, 33);
  assert.deepEqual(Object.keys(pages).sort(), Object.keys(manifest).sort());
  for (const page of Object.values(manifest)) {
    await readFile(
      new URL(`../src/pages/${page.component}.jsx`, import.meta.url),
    );
    for (const style of page.styles.filter((value) =>
      value.startsWith("/static/"),
    )) {
      await readFile(
        new URL(
          `../${style.replace("/static/", "").split("?")[0]}`,
          import.meta.url,
        ),
      );
    }
    assert.ok(
      page.fields.every(
        (field) =>
          !/(^|\.)(PasswordHash|NIN|TransactionPIN|Secret)$/.test(field),
      ),
    );
  }
});
