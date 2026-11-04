# Frontend development

All 33 application pages use React 19 and Vite. This includes authentication,
buyer and seller dashboards, marketplace, storage, chat, learning, AI tools,
orders, deliveries, wallet, purchase protection, and staff/finance administration.
Go serves the shared HTML shell, authenticated page data, static assets, and
existing form/API endpoints. URLs and session cookies remain on the same origin.

## Setup and start

Use Node.js 20.19+ or 22.12+, npm, Go 1.25+, and PostgreSQL.

```bash
npm run setup
npm start
```

Configure `backend/.env` from `backend/.env.example` first. `npm start` builds
React, then starts Go from `backend/`. Open `http://localhost:9000/` with the
example configuration, or the configured `PORT` (default `8080`).

For automatic rebuilds, run `npm run dev` from the repository root in a second
terminal. Refresh the browser after saving. This watches the Vite build; Go
continues serving the application. Restart Go after changing its page manifest.

## Code layout

- `src/pages/`: React components for every screen, loaded on demand through
  `registry.js` so a page only downloads the code it needs.
- `src/components/`: reusable navigation, authentication fields, and form controls.
- `src/hooks/`: React hooks for shared behavior.
- `src/behaviors/`: existing media, camera, wallet, and other browser integrations,
  imported as modules and mounted after React commits a page. The page lifecycle
  helper cancels timers, observers, requests, and document/window listeners.
- `src/lib/view.js`: presentation helpers for dates, status labels, and numbers.
- `css/` and `assets/`: the existing design, page styles, and images.
- `backend/internal/render/shell.html`: the single Go HTML shell.
- `backend/internal/render/pages.json`: page titles, styles, and an explicit list
  of fields Go may pass to each React screen. Collection item paths use `*`.

Navigation uses ordinary links and server form submissions. Go continues to
check permissions, validate mutations, handle multipart uploads, and redirect
after actions. React handles page rendering and shared interactive components;
browser integrations preserve the existing feature behavior. This is a React
application with a separate document for each route.

When adding page data, add the required field path to the Go manifest and use it
in the corresponding component. Do not serialize whole domain models: user models
contain private fields such as password hashes. Go computes presentation methods
and permission flags before projecting the allowed data. Bootstrap JSON is HTML
escaped and personal page responses are served with `Cache-Control: no-store`.

## Validation

```bash
npm test
```

From the repository root, this builds the browser bundles and the test-only React
renderer, runs frontend checks, then runs Go's suite. The Go page tests render
actual JSX with the exact projected data sent to browsers, retaining checks for
form fields, permissions, money actions, and escaping. Database integration tests
require the test database configuration documented by the backend.

For Chrome checks of all screens and the registration flow:

```bash
cd backend
REACT_PREVIEW_DIR=/tmp/agrilinks-react-preview go test -count=1 ./internal/handlers
cd ..
npm run test:browser
```

Set `CHROME` if Google Chrome is not at `/usr/bin/google-chrome`. Browser checks
use local fixtures and mocked API responses, including a fake camera; they do
not send SMS, create accounts, or move money. They cover mobile/desktop layouts,
navigation, login validation, password visibility, and selfie/phone/PIN signup.
Screenshots are saved to the preview directory.

## Deployment

Run `npm ci && npm run build` in `frontend/` before starting Go. Ship
`frontend/dist/`, `frontend/css/`, `frontend/assets/`, and the application's upload
storage. The Go shell and page manifest are embedded in the backend binary;
`frontend/pages/` and standalone `frontend/js/` scripts are no longer used.

Vite emits stable `dist/index.js` and `dist/index.css` entry points plus hashed
page chunks. Serve the stable entry points with revalidation. Generated bundles,
node_modules, and the test renderer are ignored by Git.
