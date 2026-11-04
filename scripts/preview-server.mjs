import { createServer } from "node:http";
import { readFile } from "node:fs/promises";
import { dirname, extname, resolve, sep } from "node:path";
import { fileURLToPath } from "node:url";

const frontend = resolve(
  dirname(fileURLToPath(import.meta.url)),
  "../frontend",
);
const types = {
  ".js": "text/javascript",
  ".css": "text/css",
  ".html": "text/html",
  ".svg": "image/svg+xml",
  ".jpeg": "image/jpeg",
  ".jpg": "image/jpeg",
  ".png": "image/png",
  ".webp": "image/webp",
};

export async function startPreviewServer(directory, aliases = {}) {
  const server = createServer(async (request, response) => {
    const path = new URL(request.url, "http://localhost").pathname;
    const base = path.startsWith("/static/") ? frontend : resolve(directory);
    const name = path.startsWith("/static/")
      ? path.slice(8)
      : aliases[path] || path.slice(1);
    const file = resolve(base, name);
    if (!file.startsWith(base + sep)) {
      response.writeHead(403).end();
      return;
    }
    try {
      response.setHeader(
        "Content-Type",
        types[extname(file)] || "application/octet-stream",
      );
      response.end(await readFile(file));
    } catch {
      response.writeHead(404).end();
    }
  });
  await new Promise((resolve, reject) => {
    server.once("error", reject);
    server.listen(0, "127.0.0.1", resolve);
  });
  return { server, baseURL: `http://127.0.0.1:${server.address().port}` };
}
