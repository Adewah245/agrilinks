// Test-only renderer used by Go's page contract tests. It exercises the actual
// JSX against the exact, filtered data that the browser receives.
import { renderToStaticMarkup } from "react-dom/server";
import { pages } from "./pages/registry.js";

let source = "";
for await (const chunk of process.stdin) source += chunk;
const match = source.match(
  /<script id="page-data" type="application\/json">([\s\S]*?)<\/script>/,
);
if (!match) throw new Error("React bootstrap data missing");
const { page, data } = JSON.parse(match[1]);
const { default: Page } = await pages[page]();
process.stdout.write(renderToStaticMarkup(<Page data={data || {}} />));
