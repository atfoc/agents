---
name: implement-ui
description: Build a working UI from a spec or description using mocked data only — never wired to a server, API, or real data source. Matches the project's existing framework and conventions, or asks which to use when the project is empty (default offer: plain HTML/CSS/JS). Wires up the real interaction logic (modals, tabs, forms, filtering, loading and error states) against an in-memory mock layer, and leaves `MOCK:` comments at every seam where real data should later be connected. Use whenever the user wants a UI built, coded, prototyped, or mocked up — "implement this screen", "build the UI for X", "make a clickable prototype", "code up this design", "mock up the front end".
---

# Implement a UI with mocked data

Build a UI that runs, looks finished, and responds to every click — with all of its data invented
locally. The end state is a UI the user can open and click through, plus a marked seam at every
point where real data must later be plugged in.

The subject is whatever the user gave you: the skill argument, a spec produced by `/create-ui`, a
design description, a screenshot, or the request earlier in the conversation. If there is no
subject, ask what screen or feature to build and stop.

## The one hard rule

**Never connect this UI to a server or a real data source.** Not "connect it behind a flag", not
"connect it if the endpoint exists". Specifically, do not write:

- `fetch`, `XMLHttpRequest`, `axios`, `WebSocket`, `EventSource`, GraphQL clients
- database drivers or ORM calls, cloud/vendor SDK calls, auth providers
- server routes, API handlers, backend files of any kind
- `.env` entries, API keys, base URLs, or config pointing at a real host
- remote image/font/script URLs (these are network calls too — see Step 6)

If the user asks mid-task to hook it up for real, say that is outside this skill and do it as
ordinary work afterwards — do not silently blend the two.

## Step 1 — Decide the technology

Look at the project before asking anything:

- Manifests: `package.json`, `pubspec.yaml`, `*.csproj`, `Cargo.toml`, `go.mod`, `Gemfile`, `composer.json`
- Frontend entry points: `index.html`, `src/main.*`, `app/`, `pages/`, `components/`
- Config: `vite.config.*`, `next.config.*`, `tailwind.config.*`, `tsconfig.json`

**If the project already has a frontend technology, use it.** Read two or three existing
components first and match what you find: component file layout and naming, styling approach
(plain CSS, CSS modules, Tailwind, styled-components, SCSS), state management, routing, TypeScript
or not, formatting. A new file should be indistinguishable from the existing ones.

Do not introduce a new framework, UI kit, or dependency. If the UI genuinely needs a library the
project lacks, ask before installing.

**If the project is empty or has no frontend**, ask the user which to use before writing anything.
Offer, in this order:

1. **Plain HTML + CSS (+ JS if needed)** — recommended default, zero build step, opens in a browser
2. **React + Vite**
3. **Whatever else fits what they described** (e.g. Vue, Svelte, Next.js)

Wait for the answer before writing files.

## Step 2 — Write the mock data layer first

All invented data lives in **its own file**, separate from the components — one place to delete
when the backend arrives. Name it by the project's convention (`src/mocks/orders.mock.js`,
`js/mock-data.js`, `lib/mock/users.ts`).

The mock module must:

- Export data whose **shape is exactly what the real API would return** — same field names, same
  nesting, same types, IDs as strings if the backend uses strings.
- Contain **realistic content**, not lorem ipsum: longest and shortest plausible strings, nullable
  fields actually null sometimes, real-looking dates and numbers, at least 8–12 list items so
  pagination and scrolling are exercised.
- Expose an **async accessor per operation**, not a bare array — `getOrders()`, `getOrder(id)`,
  `createOrder(payload)` — each returning a Promise resolved after a short artificial delay
  (200–600ms). This is what makes loading states real and makes the swap to `fetch` a one-line
  change per function.
- Keep **writes in memory** (mutate the in-module array) so create/edit/delete feel real. Note in
  a comment that a reload resets everything.
- Provide a way to reach the **failure and empty states** — e.g. an exported
  `MOCK_CONFIG = { failNext: false, emptyMode: false }` the UI or a dev control can flip.

Concrete patterns for plain JS, React, and Vue: [mock-patterns.md](mock-patterns.md).

## Step 3 — Build the UI

Implement the screen in the chosen technology, following the spec if one exists. Regardless of
stack:

- Semantic HTML: real `<button>` for actions, `<a href>` for navigation, `<label>` tied to every
  input, one `<h1>` and a sane heading order, lists as lists.
- Design tokens (CSS custom properties or the project's token system) for color, spacing, radius,
  and type — no scattered magic hex values.
- Responsive down to ~360px wide, and legible at 200% zoom.
- Visible `:focus-visible` styling on everything focusable; keyboard reachable in a logical order.
- Text contrast ≥ 4.5:1 (≥ 3:1 for large text and UI boundaries).
- Honour `prefers-reduced-motion` if you add animation.

## Step 4 — Wire every interaction for real

A mock UI is judged by whether things move. Nothing in the UI may be inert. Implement, for
whatever the screen contains:

- **Modals/popups/drawers:** open, close via button, Esc, and backdrop click; focus moves in on
  open and returns to the trigger on close; background scroll locked; `aria-modal` set.
- **Menus, dropdowns, tabs, accordions, tooltips:** open/close state, arrow-key and Esc handling,
  correct `aria-expanded` / `aria-selected` / `role`.
- **Forms:** validation on blur with inline messages, submit disabled-and-labelled while pending,
  a success result, and a failure path that keeps everything the user typed.
- **Lists:** search, filter, sort, and pagination all computed against the mock array — not fake
  controls that do nothing.
- **Loading:** skeletons or spinners shown for the duration of the mock delay, on first load and
  on refresh.
- **Empty and error states:** reachable and visually designed, not a bare "no data".
- **Toasts/confirmations:** destructive actions confirm or offer undo, and actually remove the
  item from the in-memory array.

No `href="#"` that does nothing, no button without a handler. If something is deliberately out of
scope, make it visibly non-interactive and note it in the report.

## Step 5 — Mark every seam

Every place that touches invented data gets a comment starting with the literal tag `MOCK:` so the
whole surface is greppable with `grep -rn "MOCK:"`. State what the real call should be:

```js
// MOCK: replace with `GET /api/orders?status=${status}` — expects the same Order[] shape as below.
// Delete this file once the endpoint exists; nothing else imports the raw array.
export async function getOrders(status) { ... }
```

```jsx
{/* MOCK: current user is hardcoded — read from the real session/auth context here. */}
<Avatar user={MOCK_CURRENT_USER} />
```

Tag at minimum: the mock module itself, every read, every write, hardcoded user/session/permission
values, hardcoded IDs, the artificial delay, and any feature flag or state toggle that exists only
for the demo.

Then write a short **"Replacing the mocks"** section — at the top of the mock file, or in a
`README.md` next to the UI if there is more than one mock module — listing each operation the real
backend must provide, with its method, path, and payload/response shape.

## Step 6 — Images and icons

- **Icons and simple shapes:** inline SVG or CSS. Never a remote icon-font CDN.
- **Photographs, illustrations, logos, avatars:** these need `/generate-image`. That skill is
  user-invoked only, so list the images you need — with the exact prompt and the local path each
  should be saved to — and ask the user to run it. Only run the generation yourself if they say
  to.
- **Meanwhile**, ship a local placeholder that is not a network request: a solid token-colored
  block, an inline SVG, or a small data URI, with a `MOCK:` comment naming the file that will
  replace it.
- **Never** reference `unsplash.com`, `placehold.co`, `picsum.photos`, gravatar, or any other
  remote URL — the UI must render fully offline.

## Step 7 — Run it and check

Actually open what you built — the HTML file directly, or the project's dev server (the `run`
skill covers project-specific launch commands, if one is available). Confirm the page renders, the
console is clean, and each interaction from Step 4 responds. Fix what does not work. Do not report
a UI as working without having loaded it.

## Step 8 — Report and stop

Tell the user:

1. The files you created or changed, and the technology used.
2. How to run it (exact command or file path to open).
3. The list of mock seams — what is invented and what real endpoints will be needed.
4. Anything you deliberately left non-interactive.

Then stop. Do not connect real data, do not scaffold a backend, and do not keep adding screens
that were not asked for.
