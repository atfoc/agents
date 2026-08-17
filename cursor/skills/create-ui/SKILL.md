---
name: create-ui
description: Produce a detailed UI/UX design spec for a screen, component, or flow — job story, layout, content, every state (empty/loading/error/etc.), design tokens, and accessibility notes — grounded in measurable standards (WCAG, Material, Radix) rather than adjectives like "clean" or "modern". When new imagery is needed, writes a reproducible description for each image so it can be generated later (e.g. with /generate-image); never generates the image itself. Never writes implementation code. Use whenever the user wants to plan, spec, design, or critique a UI/UX before implementation — "design a UI for X", "spec out this screen", "what should this page/flow look like", "plan the UX for X", "critique this interface".
---

# Create a UI spec

Turn a rough idea, a feature description, or an existing screen into a **design spec** — enough detail that someone (or another skill) could implement it, or critique it, without re-deciding anything. The output is prose, an ASCII wireframe, and a list of decisions. It is never code and never a generated image.

The subject is whatever the user gave you: a feature description, data to display, an existing UI to critique, or context earlier in the conversation. If there is no subject and none is inferable, ask what to design and stop.

**Out of scope — do not drift into these:**
- **No implementation code.** No HTML/CSS/JSX/React/etc. The spec is the deliverable; a separate coding skill or step consumes it.
- **No image generation.** When new imagery is needed, write the description per Step 9 and stop there. Point the user at `/generate-image` (or whatever image tool they use) to actually produce it.

Work through the steps below in order — the order is what stops the spec from jumping straight to visuals before the problem is understood.

## Step 1 — Establish the job before the layout

Write one **job story**: `When [situation], I want to [motivation], so I can [outcome].` The triggering situation drives urgency, what the user already has in hand, and what they're comparing this to. If you're working from a description rather than real research, say the job story is an approximation — don't present it as researched fact.

Then name, in one sentence each:
- **The primary action** — exactly one per view. If you can't name it, the view has no purpose yet.
- **What the user arrives with** — data, prior step, emotional state (rushed, uncertain, recovering from an error).
- **What "done" looks like for them**, not for the system.

## Step 2 — Content before layout

Design against real content, or realistic content with real length and variance — lorem ipsum hides every failure that matters. Before laying anything out, collect: longest and shortest plausible string per field; which fields can be null; realistic list lengths at both extremes; image aspect ratios and whether images can be missing.

## Step 3 — Enumerate the flow, then fill the state matrix

Walk the happy path first. Then, at each step, ask what could be missing, denied, slow, or wrong (validation failure, missing data, permission denied, timeout, server error, no results, too many results, abandon-and-return).

Then fill in the state matrix below for every state that applies to this UI. **An unlisted state is an unmade decision** — this is the single highest-value check in the whole spec.

- **Data/content:** empty-first-use (teach, don't apologize) · empty-user-cleared (confirm/congratulate — different copy from first-use) · no-results (say why + offer a way out) · exactly-one-item (does it look broken or intentional?) · many-items (pagination/virtualization/truncation, decided explicitly) · overflow (state the truncation rule) · partial/streaming (stable layout, clear end signal) · stale/cached (timestamp + refresh) · offline (what's readable/queued/blocked)
- **Loading:** initial (prefer skeletons over spinners for content-shaped regions) · incremental (preserve scroll + selection) · explicit refresh (don't discard the user's place) · timeout (say what's happening, offer cancel) · interrupted (retry resumes, doesn't restart)
- **Errors** (each needs distinct copy and distinct recovery): validation (missing/format/rejected) · server 5xx · permission 403 · not-found 404 · network failure · conflict (someone else edited) · rate limit / quota
- **Interaction** (for every interactive element): default, hover, focus-visible, active/pressed, selected, disabled, read-only, loading-while-submitting, success. `disabled` (action unavailable) and `read-only` (value shown, not editable) are different — never disable a control just to hide it, and never disable one without saying why.
- **Boundary:** RTL mirroring, localization length swing, dark mode, small screen, 200% zoom reflow, reduced motion

Output form: one line per applicable state naming its design, plus one line naming any state you deliberately excluded and why.

## Step 4 — Errors, prevention, and reversal

- Scale friction to consequence: reversible/minor → no confirmation, offer undo; destructive/permanent → explicit (ideally typed) confirmation plus a recovery window.
- Prefer undo over confirmation wherever the action is reversible — confirmation dialogs get trained away within days.
- Be liberal in what you accept as input (strip spaces, normalize dates/phone numbers/pasted currency).
- Validate on blur, not per keystroke.
- Never return an empty form on failure — preserve what the user typed.
- Error copy: what happened → why it matters (if non-obvious) → what to do next. Blame the system, never the user. Never show a raw error code as the whole message.

## Step 5 — Progressive disclosure and defaults

- A good default serves the large majority unchanged. A default users always change is a bug.
- Hidden content needs a visible door — disclosure that can't be found doesn't exist.
- Tier the interface: primary path always visible; secondary behind one interaction; rare/dangerous behind settings.

## Step 6 — Accessibility as user perspective

Write these as spec requirements ("error is announced via a live region and focus moves to the first invalid field"), not as code:
- **Keyboard-only** — sane tab order, always-visible focus, every action reachable without a pointer
- **Screen reader** — landmarks, labels tied to inputs, heading hierarchy, live regions for async updates
- **Low vision** — contrast met (see reference-standards.md), layout survives 200% zoom, no information carried by color alone
- **Motor** — target sizes (see reference-standards.md), no precision dragging without an alternative, no time pressure
- **Cognitive** — consistent navigation, plain language, no unexplained jargon
- **Situational** — glare, one-handed use, noisy room, poor connection

## Step 7 — Detect the design system, or establish one

Check in this order — first hit wins, and it's also the precedence order when sources disagree: explicit token files (`*.tokens.json`, Style Dictionary, `panda.config.ts`) → CSS custom properties in `:root` → Tailwind config (`tailwind.config.*` for v3, an `@theme` block in CSS for v4) → CSS-in-JS themes (styled-components, MUI, Chakra, Mantine) → component-library signatures (`components.json`/shadcn, Radix, antd, Bootstrap `--bs-*`) → Storybook → **inferred**: if nothing is declared, sample real components and take the mode values as the system; values off the mode are inconsistencies worth reporting in the spec.

If there's nothing to detect and nothing to infer, establish one in this order: type → spacing → color → radius/shadow/motion. Use exact numbers from [reference-standards.md](reference-standards.md) (spacing scale, type scale, motion durations, the Radix 12-step color-role table, the minimum viable token set) rather than picking values by feel.

## Step 8 — Visual craft pass

Check the layout against these amateur-vs-professional pairs and fix what you find:

- **Hierarchy** — one dominant element per view; demote the rest via size/weight/value, not decoration. Pick one emphasis channel, not several stacked. Make level differences decisive, not a 2px nudge.
- **Spacing** — vary gaps deliberately: tight within a group, loose between groups. Snap to the spacing scale.
- **Color** — near-black/near-white, not pure `#000`/`#FFF`. One accent used sparingly; neutrals do the bulk of the work (60/30/10 as a starting split). Tint neutrals toward the brand hue. Carry contrast in value, never hue alone; pair color with text/icon/shape.
- **Depth** — soft, low-opacity shadows, one consistent light direction; bigger shadow = higher elevation = more important. Never shadow and border on the same element.
- **Radius** — one radius scale applied by element size. For concentric corners: outer radius = inner radius + the padding between them.
- **Icons** — one library, one stroke weight, one corner treatment. Never emoji as functional icons.
- **Alignment** — align to a grid; optically adjust uneven shapes (play triangles, icons) after grid-aligning.

Then check against the AI-generated-look list and reject anything that appears: indigo→violet gradient · centered hero + three identical feature cards · emoji as icons · indiscriminate glassmorphism · everything symmetrical/centered/equal-weight · over-rounded corners with a shadow on every surface · unmodified framework defaults · generic copy ("Empower your workflow"). These are defaults, not decisions — wherever the brief leaves an axis free, spend that freedom on something specific to this brief.

Exact ratios, sizes, and durations (contrast, target size, motion timing, spacing/type scales) live in [reference-standards.md](reference-standards.md) — pull numbers from there rather than approximating.

## Step 9 — Describe new imagery — never generate it

First decide whether the image earns its place: yes for empty states, onboarding, error pages, explaining an abstract process, brand personality in marketing, wayfinding. No for decoration next to text that already explains itself, anything competing with the primary action, or imagery that breaks in dark mode.

For every image that earns its place, write a reproducible description covering: subject → style descriptors (line, shape, fill, detail level, perspective, texture, lighting, character, composition) → palette (actual hex values, not "brand colors") → technical specs (aspect ratio, background/transparency) → negative constraints. Read [reference-illustration.md](reference-illustration.md) for the full style vocabulary, named styles, icon-grid conventions, and set-coherence rules (a set of images must share line weight, radius, perspective, palette, detail level, and background treatment, or it reads as clip art).

Hand the finished description(s) to the user for actual generation (e.g. via `/generate-image`) — do not call an image-generation tool yourself from inside this skill.

## Step 10 — Self-critique gate

Before presenting the spec, ask: does this read as the generic thing I'd produce for any similar brief? Check specifically for the AI-generated-look items from Step 8 and for generic imagery from Step 9. State one sentence on what you changed as a result of this check — or that nothing needed to change and why you're confident of that.

## Step 11 — Cite standards carefully

Before stating any number as fact (a percentage, a conversion lift, a memory-capacity limit), check it against [reference-claims.md](reference-claims.md). Several widely-repeated UX statistics are fabricated or backwards. Prefer a mechanism you can explain over a statistic you can't source, and never present a job-story or a critique point as researched when it's actually inferred.

## Output format

**For a new design**, in this order: job story → primary action → layout structure (prose plus an ASCII wireframe) → component inventory → state matrix (Step 3) → tokens used or newly proposed (Step 7) → accessibility notes (Step 6) → image descriptions, if any (Step 9) → open questions.

**For a critique of an existing UI**, in this order: what works → ranked findings, each naming the violated principle (from Step 8, or Nielsen/Gestalt/Fitts/etc. — see [reference-standards.md](reference-standards.md) for the heuristics list) and a specific fix → what's worth checking with real users.

Present the spec directly in the conversation. Do not write it to a file unless the user asks. Stop there — do not start implementing and do not generate any images.

## Additional resources

- [reference-standards.md](reference-standards.md) — exact numbers (contrast, target size, motion, spacing/type scales, Radix color roles, minimum token set) and the heuristics/laws vocabulary (Nielsen, Gestalt, Fitts, Hick's, etc.)
- [reference-illustration.md](reference-illustration.md) — image/icon style vocabulary, named illustration styles, set-coherence rules, prompt anatomy for descriptions
- [reference-claims.md](reference-claims.md) — UX statistics that circulate widely but failed verification; check before citing a number as fact
