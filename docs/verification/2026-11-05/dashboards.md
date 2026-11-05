# Dashboard and farming assistant update

The 23 dashboard and account page types now share a green workspace palette, readable cards, responsive navigation and consistent page spacing. This includes all staff sections rendered by the administrator page, the farmer and buyer dashboards, wallet and finance pages, deliveries, saved learning plans, profile, storage, sales, orders, notifications and private security/review pages.

Account totals and shortcuts now appear near the top of the main dashboard. Existing role-dependent links, badge counts and learning guides remain connected.

The farming assistant now has a visible multiline question field, labelled photo/video and voice controls, example questions, a read-aloud option and a separate answer area. The unavailable view links to written farming guides. Existing diagnosis uploads, recorded audio, spoken replies, optional example pictures and retry behaviour remain connected to their original endpoints.

The shared page manifest loads the new styles. Restart the backend after building the frontend so the running server uses the updated embedded manifest.

## Verification

- `npm --prefix frontend test`: production and server-rendering builds and existing frontend tests passed.
- Go handler and integration checks: `TestReactPages` and `TestPublicLearningRoutesNeedNoAccountOrDatabase` passed.
- `node scripts/check-react.mjs`: all 35 page fixtures rendered at 390 and 1440 pixels, without horizontal page overflow or JavaScript exceptions. Navigation, public learning guides, topic handoff, voice input, spoken answers, example pictures, diagnosis retries, registration, storage, profile editing and wallet controls passed.
- `node scripts/check-finance-mobile.mjs`: all 28 cases passed at 320, 390, 768 and 1440 pixels. Private camera/location capture, upload failure, consent revocation, account verification, recovery and withdrawal approval gating passed.
- A focused browser check of eight synthetic farmer, buyer, staff and unavailable-assistant views passed at 390, 1024 and 1440 pixels. Staff overview, users, finance, staff permissions and settings were included. Final screenshots were inspected for spacing and contrast.

The first combined screenshot runs hit the browser time limit. The full page/interaction suite and the focused role-view suite subsequently passed in separate browser runs. The page checker now fails individual stalled browser commands promptly, skips external font/icon CDNs in previews, and uses reduced motion for deterministic screenshots.

## Live provider configuration

At verification time, `backend/.env` had an empty `GEMINI_API_KEY`, no `GOOGLE_API_KEY`, and a populated `FLUTTERWAVE_SECRET_KEY`. Neither Google key was populated in this process environment. No credentials were printed or changed.

The earlier read-only Flutterwave bank-list request returned HTTP 401. A populated setting alone does not establish provider access. Live Gemini answers and payment-provider access remain unresolved until working credentials are available to the server. The browser checks above use synthetic responses; they do not establish live provider success or execute payments.
