# Frontend development

The home page (`/`) is the first page migrated to React. Go still serves the
HTML shell in `pages/index.html` and static files under `/static/`. Login,
registration, and the remaining pages continue using their existing Go templates.

## First setup

Use Node.js 20.19+ or 22.12+ (Vite's supported versions), npm, and Go 1.25+.
Install the frontend dependencies once from the repository root:

```bash
npm run setup
```

Configure `backend/.env` using `backend/.env.example`, including your PostgreSQL
connection and SMS provider settings. To work locally without sending OTPs, use
`SMS_PROVIDER=disabled`; phone verification needs a configured provider.

## Start the frontend and backend together

From the repository root, run:

```bash
npm start
```

This builds the React page, then starts Go from `backend/` so it loads the usual
`.env` file. Go serves both the frontend and backend at the same address. The
backend only starts if the frontend build succeeds. Press Ctrl+C to stop it.

Open `http://localhost:9000/` when using the example configuration, or the port
set by `PORT`. Without a configured port, Go defaults to `8080`.

## Editing the index page

```bash
cd frontend
npm run dev
```

For automatic rebuilds while editing, run this optional watch command in a
second terminal while `npm start` is running. Refresh the browser after saving.
It does not start a separate frontend server or provide hot reload, so existing
links and sessions use the same origin. Otherwise, rerun `npm start` after edits
to rebuild and restart together.

The page is organized as follows:

- `src/main.jsx` mounts React into the HTML shell.
- `src/pages/IndexPage.jsx` assembles the home page.
- `src/components/Header.jsx` handles mobile navigation.
- `src/components/Hero.jsx` contains the introductory text and calls to action.
- `src/components/HeroSlideshow.jsx` rotates the existing farmer images, with a
  pause control and support for reduced motion.
- `src/components/Footer.jsx` contains the footer links and contact details.
- `src/styles/index.css` adds styles specific to the React page. The existing
  shared stylesheets and assets remain in use.

Navigation uses regular links to the existing Go routes. React manages the home
page's menu and slideshow; the legacy `js/navigation.js` script is only used by
the remaining HTML pages.

## Building and deploying

Run `npm ci && npm run build` from `frontend/` before starting the application,
running backend static-asset tests, or deploying. Include `frontend/dist/` with
the existing frontend files in deployments. Dependencies and generated bundles
are ignored by Git and must be built on each fresh checkout.

Vite emits stable `dist/index.js` and `dist/index.css` filenames for the Go
template. Serve these with revalidation rather than immutable cache headers.
