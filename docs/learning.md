# Learning from Agro-Shield

The public learning experience was transferred from
[Agro-Shield at 0ca84dc](https://github.com/etoosammy01/Agro-Shield/tree/0ca84dc61805dc00d4eecf2251d64f3ffca8bf93).
`GET /learning` and `POST /learning/chat` need neither an account nor a database
session. The landing page links to learning. Every dashboard includes a direct
“Learn with AI” link and a “Learn farming” shortcut.

The original learning page styling, six categories, example questions, animated
response indicator, and provider-specific teaching prompts are retained under
the Agrilinks Nigeria name. The eight written guides cover poultry, rabbits,
fish, goats, cattle, pigs, bees, and crops, with four lessons per topic. They
can be opened from the dashboard or the public learning page. A selected guide
passes its topic to AI chat. Search, empty states, keyboard dismissal, focus
restoration, and mobile layouts work in both places.

Groq is primary and OpenRouter is backup. Both default to `openai/gpt-oss-20b`,
matching the source repository. Set `GROQ_API_KEY` and `OPENROUTER_API_KEY` on
the server. `GROQ_MODEL` and `OPENROUTER_MODEL` can override their defaults.
Both providers use temperature 0.4 and low reasoning effort for GPT OSS models.
Request timeouts, token limits, and endpoints remain configurable. An error,
timeout, or empty answer from Groq falls back to OpenRouter. If neither can
answer, the question remains available for retry. Credentials stay server-side.

Public conversations stay in the current tab; changing categories or selecting
“New chat” clears the conversation. No registration is required. Saved journeys,
checklists, and learning rooms remain available through `/learn` for signed-in
users and use the same learning providers.

Verification uses local HTTP provider contract tests for both models and
failover, an account-free real-router test, and browser checks for the public
page and dashboard hub. Mocked provider/browser answers do not establish live
provider availability.
The [verification report](verification/2026-11-05/learning-ai.md) records the
current local checks and live provider access results.

## Voice and example pictures

Public chat and saved journeys accept voice questions. When available, the
browser turns speech into text; otherwise, a short recording is sent to the
public `POST /learning/transcribe` route using Groq's
`whisper-large-v3-turbo`. Questions remain editable before sending. Recordings
are limited to one minute and 2 MB. The server does not save the recordings.
Replies have a “Listen to answer” button and an optional “Read answers aloud”
setting. Speaking a question turns that setting on. Reading uses the device's
English voice, so availability and pronunciation depend on the browser.
Microphones need HTTPS (or localhost) and the person's permission.

“Show an example picture” requests an educational drawing using
`GEMINI_IMAGE_MODEL` (default `gemini-3.1-flash-lite-image`). Public learning
uses `POST /learning/example`; diagnosis uses `POST /ai-assistant/example`
and checks ownership of the saved answer. A picture failure leaves text and
spoken answers available. Pictures explain the advice and are clearly labelled
examples; they cannot confirm a disease on the person's farm.

The farming assistant sends questions and optional photos, voice recordings
and videos to `GEMINI_MODEL` (default `gemini-3.6-flash`). It accepts all 19
categories shown in the form, answers general farming questions directly, and
keeps incomplete answers out of the result. Failed requests keep the question
and attachments available for retry. Gemini's output budget has room for
reasoning while visible answers are instructed to stay under 250 words.

All teaching and diagnosis providers use shared instructions for very simple,
everyday English, short sentences, and immediate explanations of unfamiliar
words. Responses should describe what the person can see and what to do next.

Set `GEMINI_API_KEY` (or `GOOGLE_API_KEY`) on the server to enable diagnosis
and generated pictures. Keep keys out of model settings and frontend code.
Restart the backend after configuration changes. Missing keys produce honest
unavailable states. Current model names and the image Interactions request
format follow [Google's model documentation](https://ai.google.dev/gemini-api/docs/models)
and [image generation documentation](https://ai.google.dev/gemini-api/docs/image-generation).
