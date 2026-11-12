import { cpSync, mkdirSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { resolve } from "node:path";
import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";
import { backendOrigin, PREVIEW_API_PREFIX } from "./mobile/lib/url-policy.js";

const frontendDir = fileURLToPath(new URL(".", import.meta.url));

export default defineConfig(({ mode }) => {
  const environment = loadEnv(mode, frontendDir, "VITE_");
  const origin = backendOrigin(
    process.env.VITE_MOBILE_BACKEND_URL || environment.VITE_MOBILE_BACKEND_URL,
    { production: mode === "production" },
  );
  const proxy = {
    target: origin,
    changeOrigin: true,
    rewrite: (path) => path.slice(PREVIEW_API_PREFIX.length),
    configure(server) {
      server.on("proxyReq", (request) => {
        request.setHeader("Origin", origin);
        request.removeHeader("sec-fetch-site");
      });
      server.on("proxyRes", (response) => {
        if (!response.headers.location) return;
        const destination = new URL(response.headers.location, origin);
        if (destination.origin === origin)
          response.headers.location = `${PREVIEW_API_PREFIX}${destination.pathname}${destination.search}${destination.hash}`;
        else if ([301, 302, 303, 307, 308].includes(response.statusCode)) {
          response.headers["x-agrilinks-external-url"] = destination.href;
          delete response.headers.location;
          response.statusCode = 200;
        }
      });
    },
  };
  const checkPreviewOrigin = (server) =>
    server.middlewares.use(PREVIEW_API_PREFIX, (request, response, next) => {
      if (
        request.headers.origin &&
        new URL(request.headers.origin).host !== request.headers.host
      ) {
        response.statusCode = 403;
        response.end("Forbidden");
        return;
      }
      next();
    });
  return {
    root: resolve(frontendDir, "mobile"),
    envDir: frontendDir,
    base: "/",
    publicDir: false,
    resolve: { dedupe: ["react", "react-dom"] },
    plugins: [
      react(),
      {
        name: "agrilinks-mobile-assets",
        configureServer: checkPreviewOrigin,
        configurePreviewServer: checkPreviewOrigin,
        closeBundle() {
          const destination = resolve(frontendDir, "mobile-dist/static");
          for (const directory of ["css", "assets"])
            cpSync(
              resolve(frontendDir, directory),
              resolve(destination, directory),
              { recursive: true },
            );
          const icons = resolve(destination, "vendor/fontawesome");
          mkdirSync(resolve(icons, "css"), { recursive: true });
          cpSync(
            resolve(
              frontendDir,
              "node_modules/@fortawesome/fontawesome-free/css/all.min.css",
            ),
            resolve(icons, "css/all.min.css"),
          );
          cpSync(
            resolve(
              frontendDir,
              "node_modules/@fortawesome/fontawesome-free/webfonts",
            ),
            resolve(icons, "webfonts"),
            { recursive: true },
          );
        },
      },
    ],
    server: {
      host: "127.0.0.1",
      port: 5174,
      proxy: { [PREVIEW_API_PREFIX]: proxy },
    },
    preview: {
      host: "127.0.0.1",
      port: 5174,
      proxy: { [PREVIEW_API_PREFIX]: proxy },
    },
    build: {
      target: ["safari16", "chrome107"],
      cssTarget: ["safari16", "chrome107"],
      outDir: resolve(frontendDir, "mobile-dist"),
      emptyOutDir: true,
      cssCodeSplit: false,
    },
  };
});
