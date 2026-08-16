# UI/UX Design Knowledge Base

Source material for a skill that reads a rough input (data, or an idea) and outputs a **design spec and critique** for pages and components — not code.

Every claim here is marked for reliability:

- **[HARD]** — a published standard or a primary-source token value. Quote it as fact.
- **[CONVENTION]** — widely adopted practice, not mandated. Defensible as a default; say it's a convention.
- **[JUDGMENT]** — craft heuristic. True in most cases, not measurable. Never dress it as evidence.
- **[REFUTED]** — circulating design folklore that failed verification. Listed in Part 7 so it never re-enters.

The organizing principle: **an agent cannot follow an adjective.** "Make it clean" changes nothing. "One dominant element per view; demote the rest via size, weight, and value — not decoration" changes the output. Everything below is written toward the second form.

---

## Part 1 — The reasoning procedure

This part exists to stop the skill jumping straight to arranging components. The order matters: each step constrains the next, and skipping to visuals produces something that looks fine and does the wrong job.

### 1.1 Establish the job before the layout

Write a **job story**, not a persona: `When [situation], I want to [motivation], so I can [outcome].` [CONVENTION — Intercom/Klement formulation of JTBD]

It differs from a user story ("As a user, I want X") by forcing the *triggering situation* into the sentence. The situation is what determines urgency, what the user already has in hand, and what they're comparing you against.

True JTBD requires interviews. An agent working from a description is **approximating** it — say so rather than presenting an invented job story as research.

Then name, in one sentence each:

- **The primary action.** Exactly one per view. If you can't name it, the page has no purpose yet.
- **What the user arrives with** — data, context, prior step, emotional state (rushed? uncertain? recovering from an error?).
- **What "done" looks like** for them, not for the system.

### 1.2 Content before layout

Design against the **real content**, or realistic content with real length and variance. [JUDGMENT — universally endorsed in content-design practice]

Lorem ipsum hides every failure that matters: the 4-word button label that's actually "View detailed security settings and compliance reports"; the null field; the 200-character name; the list with one item and the list with 900.

Concretely, before laying anything out, collect: longest and shortest plausible string per field; which fields can be null; realistic list lengths at the extremes; image aspect ratios and whether images can be missing.

### 1.3 Enumerate the flow and its branches

Happy path first, then at each step ask: what could be missing, denied, slow, or wrong?

Standard branches to check: validation failure, missing data, permission denied, network timeout, server error, no results, too many results, user abandons mid-way and returns.

Each branch is a state that must be designed. Most are omitted in generated interfaces — which is the single most reliable difference between a demo and a product.

### 1.4 The state matrix

**The highest-value checklist in this document.** Generated UI reliably designs the success case and nothing else. Requiring this matrix is the cheapest large improvement available.

**Data / content states**
- Empty, first use (never populated) — teach, don't apologize; this is prime onboarding real estate
- Empty, user-cleared (they finished or deleted) — confirm and congratulate; *different copy from first-use*
- No results (search or filter returned nothing) — say why, and offer the way out
- Exactly one item — does the layout look broken or intentional?
- Many items — pagination, virtualization, or truncation, decided explicitly
- Overflow — long strings, huge numbers, deep nesting; state the truncation rule
- Partial / streaming — stable layout, no reflow thrash, a clear end signal
- Stale / cached — timestamp and refresh affordance
- Offline — what's readable, what's queued, what's blocked

**Loading states**
- Initial load — prefer skeletons over spinners for content-shaped regions [CONVENTION]
- Incremental (pagination, append) — preserve scroll position and selection
- Explicit refresh — don't discard the user's place
- Timeout — say what's happening and offer cancel
- Interrupted mid-load — retry must resume, not restart

**Error states** — each needs distinct copy and distinct recovery
- Validation: required missing / format wrong / value rejected
- Server (5xx), permission denied (403), not found (404)
- Network failure, conflict (someone else edited), rate limit, quota exceeded

**Interaction states** — for every interactive element
- default, hover, focus-visible, active/pressed, selected, disabled, read-only, loading-while-submitting, success

Note: `disabled` and `read-only` are different. Disabled = action unavailable. Read-only = value shown, editing not offered. Never disable a control to hide it, and never disable without explaining why.

**Boundary conditions**
- RTL mirroring, localization length swing, dark mode, small screen, 200% zoom reflow, high-contrast mode, reduced motion

**Output form for a spec:** list every applicable state with a one-line description of its design, then a line naming any state deliberately excluded and why. An unlisted state is an unmade decision.

### 1.5 Errors, prevention, and reversal

- **Scale friction to consequence.** [JUDGMENT] Reversible and minor → no confirmation, offer undo. Destructive and permanent → explicit confirmation, ideally typed, plus a recovery window.
- **Prefer undo to confirmation** for anything you can reverse. Confirmation dialogs are trained away within days; undo works even when the user was on autopilot.
- **Be liberal in what you accept.** Strip spaces, normalize dates and phone numbers, accept the paste with the currency symbol in it. (Postel's principle, applied to input — note it's contested as a *protocol* rule for security reasons, but sound for human-facing form input.)
- **Validate on blur, not per keystroke.** Don't show an error for an email the user is still typing.
- **Preserve input on failure.** Never return an empty form.
- **Error copy formula:** what happened → why it matters (if non-obvious) → what to do next. Blame the system, not the user. Never surface a code as the whole message.

### 1.6 Progressive disclosure and defaults

- A good default serves the large majority without being changed. A default users always change is a design bug.
- Hidden content needs a **visible door**. Progressive disclosure fails silently when the disclosure control isn't findable.
- Tier the interface: primary path always visible; secondary behind one interaction; rare and dangerous behind settings.

### 1.7 Accessibility as user perspective, not compliance

Design the spec against people, and the checks follow naturally:

- **Keyboard-only** — is tab order sane, is focus always visible, can every action be reached without a pointer?
- **Screen reader** — landmarks, labels tied to inputs, heading hierarchy, live regions for async updates
- **Low vision** — contrast ratios met, layout survives 200% zoom, information never carried by color alone
- **Motor** — target sizes, no precision dragging without an alternative, no time pressure
- **Cognitive** — consistent navigation, plain language, no unexplained jargon
- **Situational** — glare, one hand, noisy room, poor connection

State these as spec requirements, not code. "Error is announced via a live region and focus moves to the first invalid field" belongs in a spec; the `aria-live` attribute belongs in the implementation.

---

## Part 2 — The measurable standards

The numbers that turn "this feels cramped" into a claim someone can check.

### 2.1 Contrast — WCAG 2.2 [HARD]

| Requirement | Ratio | Criterion |
|---|---|---|
| Normal text, AA | 4.5:1 | 1.4.3 |
| Large text, AA | 3:1 | 1.4.3 |
| UI components & graphical objects | 3:1 | 1.4.11 |
| Normal text, AAA | 7:1 | 1.4.6 |
| Large text, AAA | 4.5:1 | 1.4.6 |

**"Large text" means ≥18pt (24px), or ≥14pt bold (18.5px).** [HARD]

APCA / WCAG 3 is still draft and perceptually calibrated rather than ratio-based; don't cite it as a requirement yet, but note that Radix Colors already targets APCA Lc values.

### 2.2 Text spacing — WCAG 2.2 SC 1.4.12 [HARD]

Content must survive a user overriding to: line height **1.5×** font size; paragraph spacing **2×** font size; letter spacing **0.12×**; word spacing **0.16×**.

### 2.3 Target size [HARD, with a real conflict]

| Source | Minimum | Note |
|---|---|---|
| WCAG 2.5.8 (AA) | **24×24 CSS px** | with spacing exceptions |
| WCAG 2.5.5 (AAA) | **44×44 CSS px** | enhanced |
| Apple HIG | **44×44 pt** | long-standing iOS guidance |
| Material Design | **48×48 dp** | de facto Android standard |

They genuinely disagree. Sensible default: **44×44** for touch, never below 24×24 for any pointer target. Note the units differ (CSS px / pt / dp) and are not interchangeable at arbitrary densities.

### 2.4 Motion — Material Design 3 primary token values [HARD]

Retrieved from the published token source (`_md-sys-motion.scss`, v0.192):

| Band | Values (ms) |
|---|---|
| short1–4 | 50, 100, 150, 200 |
| medium1–4 | 250, 300, 350, 400 |
| long1–4 | 450, 500, 550, 600 |
| extra-long1–4 | 700, 800, 900, 1000 |

Easing tokens: `standard` and `emphasized` = `cubic-bezier(0.2, 0, 0, 1)`; `emphasized-accelerate` = `cubic-bezier(0.3, 0, 0.8, 0.15)`; `emphasized-decelerate` = `cubic-bezier(0.05, 0.7, 0.1, 1)`.

**Working defaults:** 150–200ms for state changes, 200–300ms for standard transitions, **400ms as the hard ceiling for anything user-triggered.** The whole published scale stops at 1s, and that upper band is for large orchestrated sequences, not micro-interactions.

The 400ms ceiling coincides with the **Doherty threshold** (Doherty & Thadani, IBM, 1982) — response under ~400ms keeps the user in flow. Two independent sources landing on the same number is a good sign; treat it as the real boundary.

**Response-time perception** (Nielsen, from Miller 1968): **0.1s** feels instantaneous; **1s** keeps thought uninterrupted; **10s** is the limit of attention — past it, users switch away and you need a progress indicator. [HARD]

Always honor `prefers-reduced-motion`.

### 2.5 State layers and elevation — Material 3 [HARD]

State layer opacity: hover **0.08**, focus **0.12**, pressed **0.12**, dragged **0.16**. Elevation levels: 0, 1, 3, 6, 8, 12dp.

### 2.6 Typography [CONVENTION unless noted]

**Modular scale ratios:** 1.067 minor second · 1.125 major second · 1.2 minor third · 1.25 major third · 1.333 perfect fourth · 1.414 augmented fourth · 1.5 perfect fifth · 1.618 golden.

Tighter ratios (1.125–1.25) suit dense product UI; wider (1.333+) suit marketing pages where display type needs to dominate.

**Line length:** 45–75 characters is the readability target; WCAG 1.4.8 caps at 80 (40 for CJK) [HARD for the WCAG cap].

**Line height:** 1.4–1.6 for body, tighter (≈1.2) for large headings. WCAG's 1.5 minimum applies to the user-override requirement, not to every element.

**Body size:** 16px is the practical web default; below 14px is hard to defend for body copy. No WCAG mandate exists for a minimum size.

**Restraint:** one or two typefaces; two sizes and two weights will carry most hierarchy. Reach for color and spacing before adding a third size. [JUDGMENT — this is the single highest-leverage typographic rule]

**All-caps** needs 5–12% letter-spacing to stay readable, and belongs in small labels, not body text.

### 2.7 Spacing [CONVENTION]

4px or 8px base unit. A usable scale: **4, 8, 12, 16, 24, 32, 48, 64**. The steps grow non-linearly on purpose — adjacent values must be distinguishable, or the scale stops encoding anything.

**The rule that matters more than the scale: space encodes relationship.** Related things sit closer than unrelated things. Uniform padding everywhere destroys grouping and is the most common spacing failure.

### 2.8 Layout [CONVENTION]

Breakpoints vary by system (Bootstrap: 576/768/992/1200/1400). Adopt the project's, or pick three. WCAG 1.4.10 requires reflow without horizontal scrolling at **320px** equivalent [HARD]. 12-column grids are the common default. Measure-driven max width for text: roughly 600–900px.

### 2.9 Performance as UX — Core Web Vitals [HARD]

| Metric | Good | Needs work | Poor |
|---|---|---|---|
| LCP | ≤2.5s | 2.5–4s | >4s |
| INP | ≤200ms | 200–500ms | >500ms |
| CLS | ≤0.1 | 0.1–0.25 | >0.25 |

Measured at the 75th percentile. CLS is a *design* failure as much as an engineering one — reserve space for images, embeds, and async content in the spec.

---

## Part 3 — Visual craft: professional vs. amateur

Each entry is checkable against a screenshot. That's the bar for inclusion.

### 3.1 Hierarchy

- **Everything at equal weight.** The eye has nowhere to land. → One dominant element per view; demote the rest. Reduce the secondary rather than amplifying the primary.
- **Emphasis by accumulation** — bold *and* larger *and* colored *and* boxed. → Pick one channel.
- **Insufficient contrast between levels.** A 2px size difference reads as a mistake. → Make steps decisive.

### 3.2 Spacing and rhythm

- **Uniform gaps between everything.** Destroys grouping. → Vary deliberately; tight within a group, loose between groups.
- **Padding not scaled to density.** Dashboards need compact padding; marketing needs air. Applying one value everywhere yields either cramped or hollow.
- **Off-scale values** (13px, 27px). → Snap to the scale.

### 3.3 Color

- **Pure `#000` on pure `#FFF` with `#ccc` borders.** Reads as unstyled. → Near-blacks, near-whites, low-contrast borders or none. *(Keep this rule — but see Part 7: the usual reading-speed justification is false. The real reasons are fatigue over long sessions and visual refinement.)*
- **Too many hues at full saturation.** → One accent, used sparingly; neutrals do the bulk of the work. The 60/30/10 split is a useful starting heuristic. [JUDGMENT]
- **Untinted grays** next to a saturated brand color look dead. → Tint neutrals slightly toward the brand hue.
- **Hue-only contrast.** Fails in grayscale and for color-blind users. → Check in grayscale; carry contrast in *value*.
- **Color as sole carrier of meaning.** → Always pair with text, icon, or shape. [HARD — WCAG 1.4.1]

### 3.4 Depth

- **Harsh, uniform, pure-black shadows on everything.** → Soft, low-opacity, consistent light direction (pick one, usually from above). Bigger shadow = higher elevation = more important; if elevation doesn't track importance, remove it.
- **Shadow and border on the same element.** → Choose one.

### 3.5 Radius

- **Inconsistent radii** across the interface. → One radius scale, applied by element size.
- **Nested radius error.** For concentric corners to look right: **outer radius = inner radius + padding between them.** Using the same value on both makes the outer corner look pinched. [CONVENTION — the geometric relationship is real; the commonly-quoted form of this rule is often garbled, so state it in this direction.]

### 3.6 Icons

- **Mixed sets or mixed weights.** Immediately visible. → One library, one stroke weight, one corner treatment.
- **Scaling without adjusting stroke.** Strokes go muddy small, spindly large.
- **Emoji as functional icons.** Can't inherit color, won't adapt to theme, no states. Also a loud AI tell. → Use a real icon set.

### 3.7 Alignment

- **Near-alignment.** A few pixels off reads as sloppy even when the viewer can't name why. → Align to a grid.
- **Mathematical centering of optically-uneven shapes** (play triangles, icons with uneven mass). → Adjust by eye after aligning by grid. Optical correction is a real technique amateurs skip.

### 3.8 The AI-generated look

Specific, current tells worth naming explicitly so the skill can reject them:

- **Indigo→violet gradient** (roughly `#4F46E5`→`#8B5CF6`). Traceable to a framework default; now a training-data feedback loop.
- **Centered hero + three identical feature cards.** The statistical average of every generated landing page.
- **Emoji standing in for an icon set.**
- **Glassmorphism applied indiscriminately**, usually at the cost of contrast.
- **Everything symmetrical, everything centered, every card identical weight.**
- **Over-rounded corners and a shadow on every surface.**
- **Unmodified framework defaults** — default palette, default font, default spacing.
- **Generic copy** — "Empower your workflow," "Seamlessly integrate."

The general principle: these are **defaults, not decisions**. A default isn't wrong because it's ugly; it's wrong because nothing about it responds to *this* brief. Where the brief leaves an axis free, spend that freedom on something the subject actually implies.

### 3.9 What professionals do that amateurs omit

Optical alignment after grid alignment · neutrals tinted toward the brand hue · one accent used sparingly · contrast carried by value, not hue · strict spacing-scale discipline · one deliberate focal point · hierarchy from two sizes and two weights, not six · space instead of borders for grouping · one committed light direction · deliberate asymmetry.

---

## Part 4 — Detect the system, or establish one

The skill runs in two modes. Deciding which one it's in is the first branch.

### 4.1 Detection — where a design system hides

Check in this order; **first hit wins**, because this is also the precedence order when sources disagree:

1. **Explicit token files** — `*.tokens.json` (W3C DTCG format, first stable version October 2025; `$value` / `$type` / `$description`), Style Dictionary `config.json`, `panda.config.ts`
2. **CSS custom properties** in `:root` — `--color-*`, `--space-*`, `--radius-*`, `--shadow-*`, plus dark-mode overrides under `.dark` or `prefers-color-scheme`
3. **Tailwind** — `tailwind.config.{js,ts}` for v3. **For v4, theme moved into CSS** under an `@theme` block alongside `@import "tailwindcss"`. Detection logic written only against v3 will miss v4 projects entirely.
4. **CSS-in-JS themes** — styled-components / Emotion `ThemeProvider`, MUI `createTheme()`, Chakra `extendTheme()`, Mantine `MantineProvider`
5. **Component library signatures** — `components.json` (shadcn/ui, note `cssVariables: true` and that components are *copied into the repo*, so they're editable and may already have drifted); `@radix-ui/*` packages; `antd` with `ConfigProvider`; Bootstrap's `--bs-*` properties
6. **Storybook** — `.storybook/`, `*.stories.tsx`, possibly a design-token addon
7. **Inferred de facto system** — when nothing is declared, sample real components: histogram the padding, radii, shadow, and font-size values actually in use. The mode values *are* the system. Values off the mode are inconsistencies worth reporting.

**Report what you found and what it implies**, including detected inconsistency. "The declared scale is 4/8/16 but 6px and 10px appear in 14 components" is a genuinely useful critique finding.

### 4.2 Establishing from scratch

Decision order: **type → spacing → color → radius/shadow/motion.** Type and spacing determine structure; color is applied to structure that already works.

**Color ramps.** Do not step HSL lightness evenly — perceived lightness diverges badly across hues (yellow blows out, blue goes muddy). Use a perceptually uniform space: **OKLCH** (step L evenly, hold C and H) or Material's **HCT** tonal palettes.

**The single best shortcut: adopt Radix Colors' 12-step scale**, which assigns a semantic role per step:

| Steps | Role |
|---|---|
| 1–2 | App and subtle backgrounds |
| 3–5 | Component backgrounds: normal, hover, active |
| 6–8 | Borders: subtle, standard, strong / focus ring |
| 9–10 | Solid fills (9 is the pure brand color), hover |
| 11–12 | Text: low-contrast and high-contrast |

Steps 11 and 12 are built to hit APCA Lc 60 and Lc 90 against step 2. Adopting this means color decisions become *lookups* rather than judgments — which is exactly what makes it work well in a skill.

**Neutrals.** Pure gray reads cold and clinical. Tint slightly toward the brand hue (or deliberately warm/cool) — a small saturation, consistently applied.

**Dark mode is not an inversion.** [CONVENTION, strongly held across systems] Build a separate ramp. In dark mode, elevation is expressed by *lighter* surfaces rather than shadows; saturated colors need desaturating to avoid vibration against dark backgrounds.

**Minimum viable token set** — the smallest complete system:
- ~15 colors (primary, secondary, 4 semantic, 6–10 neutrals)
- 8 spacing steps
- 4–5 type sizes, 2–3 weights, 2–3 line heights
- 4 radii
- 3 shadows
- 2–3 motion durations + easings
- 3 breakpoints
- 5–7 z-layers

**Worth adopting wholesale:** Radix Colors (palettes), Open Props (full CSS-variable set), Tailwind's default theme, USWDS or GOV.UK (accessibility-first, government-tested).

---

## Part 5 — Illustration and imagery

### 5.1 When it earns its place

NNGroup eye-tracking finds users attend closely to **task-relevant** images and learn quickly to ignore **decorative** ones, spending their fixations on text instead. *(Precise wording matters — see Part 7. The often-repeated "zero fixations" is an overstatement of what NNGroup actually claims.)*

**Earns it:** empty states · onboarding · error pages · explaining an abstract process · brand personality in marketing · wayfinding.

**Doesn't:** generic decoration next to text that already explains itself · anything competing with the primary action · imagery that breaks in dark mode · mixed styles across one product.

### 5.2 The style specification vocabulary

To be reproducible, a style must be *described*, not gestured at. The axes:

- **Line** — stroke present or not; weight; cap and join; hard vs. textured edge
- **Shape** — geometric vs. organic; grid-constructed vs. freehand
- **Fill** — palette size; relationship to brand color; flat vs. gradient
- **Detail level** — readable at 16px / 32px / 128px+
- **Perspective** — flat, isometric (30°), 2.5D, linear perspective, orthographic
- **Texture** — clean vector, grain, halftone, painterly
- **Lighting** — direction, shadow type, highlights present or not
- **Character** — faceless / simplified / detailed; proportions; expression range; diversity
- **Composition** — centering, negative space ratio, background treatment, crop

### 5.3 Set coherence — the axes that must hold

Line weight · corner radius · perspective · palette · detail level · character proportions · prop scale · background treatment · texture presence · shape language.

Break any one across a set and the set reads as clip art. The cheapest guarantee of coherence is **one source** — a single library or a single style reference.

### 5.4 Named styles

**Flat** — single plane, 2–5 solid colors, no depth. Scales, recolors, and loads well; safe but generic.
**Corporate Memphis / Alegria** — flat with exaggerated limbs, small heads, faceless figures, non-representational skin tones. Heavily oversaturated and now widely criticized; using it signals "default" more than "friendly."
**Isometric technical** — 30° axes, true scale, no vanishing point. Best for explaining systems and processes.
**Hand-drawn / textured** — visible stroke, organic line, grain. Warm and distinctive; hardest to keep consistent.
**3D / claymorphism** — soft-body forms, pastel palettes, ambient light. Playful; showing fatigue.
**Line art** — stroke-only, consistent weight. The dominant modern UI idiom.
**Spot illustration** — small, single-idea, 100–300px, high contrast for small-scale legibility.

### 5.5 Icons

Icons are not small illustrations. They obey a grid system: typically 24×24 with a ~20×20 live area, keyline shapes (square ≈18, circle ≈20) that equalize *optical* size across differently-shaped glyphs, and one stroke weight (commonly 2dp/2px) held across the entire set.

| Library | Grid | Weight | Character |
|---|---|---|---|
| Lucide / Feather | 24 | 2px | Minimal, slightly rounded; the modern default |
| Heroicons | 24 (+ sizes) | varies | Outline/solid/mini; Tailwind-native |
| Phosphor | 24 | 6 weights | Thin→Bold, Fill, Duotone; most flexible |
| Material Symbols | variable | variable | Outlined/Rounded/Sharp; variable font |
| Radix | 15 | light | Tiny, crisp, precise |
| Tabler | 24 | adjustable | Largest free set |

Use `currentColor` so icons inherit text color and follow the theme automatically. Icon-only controls need an accessible name; icon+label is safer for anything not universally understood.

### 5.6 Generation

**Prompt anatomy:** subject → style descriptors → composition → palette (with actual hex values) → technical specs (aspect ratio, background/transparency) → negative constraints.

**Consistency across a set**, in order of effectiveness:
1. A **style reference image** — generate one you like, then reference it. Solves most drift.
2. A **fixed style block** pasted into every prompt, varying only the subject line.
3. **Seed control**, where supported.
4. Prompt discipline — short and non-contradictory beats long and hedged.

*(Tool-specific flags for reference-image and seed control change frequently; verify against current docs rather than trusting a remembered flag name.)*

**Failure modes:** character drift across a set · garbled text inside images (never ask for readable text — composite it afterward) · uncanny proportions · palette drift · unusable backgrounds · wrong aspect ratio · style mixing within one image.

**Licensing:** terms differ by model and tier and training-data provenance is contested. For anything commercial, verify rather than assume.

**Open sets worth knowing:** unDraw (CC0, recolorable), Humaaans (CC0, composable), Open Peeps (CC0), DrawKit (MIT tier).

### 5.7 SVG and delivery

Prefer SVG for icons, diagrams, and geometric illustration: it scales, themes via `currentColor`, animates, and stays small. Prefer raster (WebP with PNG fallback) for photographic or highly detailed organic work, where SVG path count explodes.

Always set `viewBox` and let CSS control size. For dark mode, `currentColor` or CSS custom properties beat shipping two files. Optimize with SVGO.

**Alt text:** decorative → `alt=""` (and `aria-hidden`), so screen readers skip it. Informative → describe the *information*, not the artwork. "Diagram: three clients connecting to one server" — not "isometric illustration in blue and green."

---

## Part 6 — Heuristics reference

Useful for critique vocabulary. Naming the violated principle makes a critique reviewable rather than a matter of taste.

**Nielsen's 10 usability heuristics** — visibility of system status · match to the real world · user control and freedom · consistency and standards · error prevention · recognition over recall · flexibility and efficiency · aesthetic and minimalist design · help users recover from errors · help and documentation.

**Shneiderman's 8 golden rules** — consistency · universal usability · informative feedback · dialogs that yield closure · error prevention · easy reversal · user control · reduced short-term memory load.

**Gestalt** — proximity · similarity · closure · continuity · figure/ground · common region · common fate · uniform connectedness · Prägnanz. These are the mechanism behind most spacing and grouping critique: "these two cards read as one group because proximity overrides the border" is a precise, checkable observation.

**Norman** — affordances · signifiers · constraints · mappings · feedback · conceptual model. Best tool for diagnosing *why* something is confusing rather than just that it is.

**Laws worth applying:**
- **Fitts's** — acquisition time grows with distance, shrinks with target size. Screen edges and corners are effectively infinite targets. Put frequent actions near where the pointer already is.
- **Hick's** — decision time grows logarithmically with the number of options. Chunking reduces effective *n*. Note the logarithm: doubling options does *not* double decision time.
- **Jakob's** — users spend most of their time on other products and bring those expectations with them. Deviate deliberately, not accidentally.
- **Doherty threshold** — ~400ms. Already load-bearing in Part 2.
- **Von Restorff** — the item that differs is remembered. The mechanism behind a single accent color.
- **Serial position** — first and last items are recalled best. Order menus accordingly.
- **Peak–end** — experiences are remembered by their peak and their ending, not their average.
- **Aesthetic–usability effect** (Kurosu & Kashimura, 1995) — attractive interfaces are *perceived* as more usable and buy tolerance for minor flaws. It buys tolerance; it does not fix real problems.
- **Goal-gradient** — effort increases near a visible goal; endowed progress amplifies it. The basis for progress indicators.
- **Tesler's** — complexity is conserved. It moves between user and implementation; it doesn't vanish. Useful for arguing that a simple UI requires more engineering, not less.

**Handle with care:** Miller's 7±2 (see Part 7) · Zeigarnik effect (2025 meta-analyses find no reliable recall advantage; the related Ovsiankina resumption effect holds up better) · Postel's law (sound for form input, contested as a protocol rule).

---

## Part 7 — The claims ledger

**This part is defensive.** Every item below circulates widely and failed verification. A skill that repeats them will produce confident, checkable-and-wrong rationale — worse than giving no rationale at all.

| Claim | Verdict | What's actually true |
|---|---|---|
| "Cramped screens → users abandon 47% faster" | **Fabricated** | Traces to Gloria Mark's finding that average attention *per screen* is ~47 **seconds**. A duration was converted into a fake percentage. |
| "Generous spacing boosts conversions 28%" | **Unsupported** | No traceable study. Whitespace does help; documented effects range ~20–35% depending on metric and context. Don't quote a single figure. |
| "Pure black on white cuts reading speed 20%" | **Backwards** | Piepenbrock, Mayr & Buchner (2014): positive polarity (black on white) produced *better* proofreading speed and accuracy (d=0.68, d=0.77). The ~20% figure describes *reduced* contrast. **Keep the near-black convention — justify it by long-session visual fatigue and refinement, not legibility.** |
| "Recognition 85–95% vs recall 35–50%" | **Directionally right, falsely precise** | Recognition reliably beats recall (Yonelinas; Karpicke & Roediger 2008; Tulving & Pearlstone). The exact percentages are an amalgam, not one study. State the direction, not the digits. |
| "Max 7 menu items (Miller's law)" | **Contradicted** | Miller (1956) concerned absolute judgment of unidimensional stimuli and immediate serial recall — not visible interface options. He himself called the recurring seven "a pernicious, Pythagorean coincidence." Cowan (2001) puts working memory near **4** chunks. Critically, **visible menus are a recognition task, not a memory task.** Landauer & Nachbar (1985) tested breadth of 2/4/8/16 across 4,096 choices and found **broader, shallower menus outperform deeper ones**; Kiger (1984) suggested 8–9 per level. **A 7-item cap forces deeper hierarchies and makes things worse.** |
| "Never nest deeper than 3 levels" | **Partly right, wrong shape** | Depth does degrade performance, but gradually — there's no validated cliff at 3. Defensible: prefer breadth over depth; 2–3 levels ideal; watch closely past 4, especially on mobile. |
| "Decorative images get zero eye fixations" | **Overstated** | NNGroup describes the behavior categorically — users *ignore* decorative images and concentrate fixations on text — without quantifying it as zero. Say "substantially fewer," not "zero." |

**The transferable lesson:** design writing recirculates invented precision. A number with no traceable study is a red flag, and oddly specific numbers (47%, 28%) are the most suspicious of all. Prefer a mechanism you can explain over a statistic you can't source.

---

## Part 8 — Notes for turning this into a skill

Derived from this project's `create-claude-skill/SKILL.md` and from how shipped design skills are actually built.

**Hard constraints from the project's own rules:**
- SKILL.md under **~500 lines**; overflow goes to sibling reference files that load only when the body points at them
- `description` frontmatter *is* the routing decision — name the trigger in the user's vocabulary plus the action; `description` + `when_to_use` capped at 1,536 chars
- **Say what to do, not why.** Cut rationale unless the rule fails without it
- Make rules checkable; state the stop condition; say what the output is and where it goes
- Body must work standalone, with no conversation context

**What makes design guidance actually bite:**
1. **Procedure, not principles.** Numbered steps in a forced order. The order is the value — it's what stops the model going straight to visuals.
2. **Concrete defaults over instructions to choose.** Ship an actual palette, an actual scale, actual durations. "Pick a good palette" is inert; Radix's 12 steps with assigned roles is executable.
3. **Named anti-patterns.** Part 3.8 exists so the skill can reject its own most likely output. This is the highest-value section for output quality, because the failure mode isn't ignorance — it's defaults.
4. **Forced self-critique.** A second pass that asks "does this read as the generic thing I'd produce for any similar brief?" — and requires stating what changed. `frontend-design` does exactly this and it's the mechanism most worth copying.
5. **Checklists that produce visible omissions.** The state matrix works because an unlisted state is a visible gap in the output.
6. **Thresholds over adjectives** everywhere a number exists.

**Suggested split:**

- **SKILL.md** — the procedure (Part 1), the mode branch (Part 4.1 detect vs. 4.2 establish), the critique pass (Part 3), the self-critique gate, and the output format. Pointers to references.
- **`reference-standards.md`** — Part 2 numbers plus Part 6 heuristics
- **`reference-illustration.md`** — Part 5
- **`reference-claims.md`** — Part 7, loaded whenever the skill is about to cite evidence for a recommendation

**Define the output format explicitly.** Since this produces specs and critique rather than code, the format *is* the deliverable. Suggested spec shape: job story → primary action → layout structure (prose + ASCII wireframe) → component inventory → state matrix → tokens used or proposed → accessibility notes → open questions. Suggested critique shape: what works → ranked findings, each with the violated principle and a specific fix → what to check with real users.

**Design for composition.** This skill's output can feed `frontend-design`, which turns a brief into HTML/CSS. Emitting a spec in the shape that skill wants as input makes the pair work as a pipeline.
