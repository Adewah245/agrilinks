# Learning, voice, pictures and diagnosis verification

Verified on 2026-11-05.

The frontend production build and rendering tests pass. Configuration and
service tests pass. Go vet passes for config, services, handlers and integration.
The real router and disposable PostgreSQL checks pass for saved learning
journeys, all 19 diagnosis categories, JSON success/failure, diagnosis picture
ownership, and public learning, voice and picture endpoints without registration.

Local provider tests verify both original learning models and failover,
simple-English instructions, Gemini photo/audio/video requests, complete
visible text, caller cancellation, thinking settings, image generation request
format and response validation, and Groq transcription uploads and failures.
No live provider keys are used by these tests.

The browser script passes all 70 layouts (35 pages at 390px and 1440px).
Learning interaction checks pass for guide search, four lessons per guide,
Escape and focus restoration, topic handoff, conversation history, error
retries, safe answer rendering, voice typing, real browser audio recording
using a synthetic microphone, transcription, discarded recordings, spoken
reply controls, example picture failures/retries and diagnosis audio submission.
Speech recognition, speech synthesis, transcription and AI responses are mocked;
these checks do not prove real speech accuracy or live provider availability.
Existing registration, selfie, storage, profile and wallet browser checks pass.

Live checks sent one short farming question to each configured learning
provider: Groq returned HTTP 403 and OpenRouter returned HTTP 401. Gemini has
no key configured locally or in the process environment, so live Gemini
diagnosis and image generation could not be verified.

Before live use, configure a valid `GEMINI_API_KEY` (or `GOOGLE_API_KEY`) and
resolve Groq/OpenRouter credential or account access in `backend/.env`, then
restart the backend. Keys must stay in server configuration. The current
implementation retains both providers; the refused live requests have not
been reported as successful AI replies.
