# UI/UX measurable standards and heuristics reference

Read this when Step 7 or Step 8 of `SKILL.md` needs an exact number, or when Step 11 needs a named principle to cite in a critique. Reliability tags: **[HARD]** published standard or primary-source token value, quote as fact · **[CONVENTION]** widely adopted but not mandated, defensible as a default · **[JUDGMENT]** craft heuristic, true in most cases, not measurable — never dress it as evidence.

## Contrast — WCAG 2.2 [HARD]

| Requirement | Ratio | Criterion |
|---|---|---|
| Normal text, AA | 4.5:1 | 1.4.3 |
| Large text, AA | 3:1 | 1.4.3 |
| UI components & graphical objects | 3:1 | 1.4.11 |
| Normal text, AAA | 7:1 | 1.4.6 |
| Large text, AAA | 4.5:1 | 1.4.6 |

"Large text" means ≥18pt (24px), or ≥14pt bold (18.5px). APCA/WCAG 3 is still draft — don't cite it as a requirement, but note Radix Colors already targets APCA Lc values.

## Text spacing — WCAG 2.2 SC 1.4.12 [HARD]

Content must survive a user override to: line height 1.5× font size · paragraph spacing 2× font size · letter spacing 0.12× · word spacing 0.16×.

## Target size [HARD, with a real conflict between sources]

| Source | Minimum | Note |
|---|---|---|
| WCAG 2.5.8 (AA) | 24×24 CSS px | with spacing exceptions |
| WCAG 2.5.5 (AAA) | 44×44 CSS px | enhanced |
| Apple HIG | 44×44 pt | long-standing iOS guidance |
| Material Design | 48×48 dp | de facto Android standard |

Sensible default: 44×44 for touch, never below 24×24 for any pointer target. Units (px/pt/dp) are not interchangeable at arbitrary densities.

## Motion — Material Design 3 tokens [HARD]

| Band | Values (ms) |
|---|---|
| short1–4 | 50, 100, 150, 200 |
| medium1–4 | 250, 300, 350, 400 |
| long1–4 | 450, 500, 550, 600 |
| extra-long1–4 | 700, 800, 900, 1000 |

Easing: `standard`/`emphasized` = `cubic-bezier(0.2, 0, 0, 1)`; `emphasized-accelerate` = `cubic-bezier(0.3, 0, 0.8, 0.15)`; `emphasized-decelerate` = `cubic-bezier(0.05, 0.7, 0.1, 1)`.

Working defaults: 150–200ms for state changes, 200–300ms for standard transitions, **400ms hard ceiling for anything user-triggered** — this coincides with the Doherty threshold (response under ~400ms keeps the user in flow). Response-time perception (Nielsen/Miller): 0.1s feels instantaneous, 1s keeps thought uninterrupted, 10s is the attention limit — past it, show a progress indicator. Always honor `prefers-reduced-motion`.

## State layers and elevation — Material 3 [HARD]

State layer opacity: hover 0.08, focus 0.12, pressed 0.12, dragged 0.16. Elevation levels: 0, 1, 3, 6, 8, 12dp.

## Typography [CONVENTION unless noted]

Modular scale ratios: 1.067 minor second · 1.125 major second · 1.2 minor third · 1.25 major third · 1.333 perfect fourth · 1.414 augmented fourth · 1.5 perfect fifth · 1.618 golden. Tighter (1.125–1.25) suits dense product UI; wider (1.333+) suits marketing/display type.

Line length: 45–75 characters is the readability target; WCAG 1.4.8 caps at 80 (40 for CJK) [HARD for the cap]. Line height: 1.4–1.6 body, ~1.2 for large headings. Body size: 16px is the practical web default; below 14px is hard to defend (no WCAG minimum exists). Restraint: one or two typefaces, two sizes and two weights carry most hierarchy — reach for color/spacing before a third size [JUDGMENT, highest-leverage typographic rule]. All-caps needs 5–12% letter-spacing and belongs in small labels, not body text.

## Spacing [CONVENTION]

4px or 8px base unit. Usable scale: 4, 8, 12, 16, 24, 32, 48, 64 — steps grow non-linearly so adjacent values stay visually distinguishable. The rule that matters more than the scale: **space encodes relationship** — related things sit closer than unrelated things; uniform padding everywhere destroys grouping.

## Layout [CONVENTION]

Breakpoints vary by system (Bootstrap: 576/768/992/1200/1400) — adopt the project's or pick three. WCAG 1.4.10 requires reflow without horizontal scrolling at 320px equivalent [HARD]. 12-column grids are the common default. Measure-driven max width for text: roughly 600–900px.

## Performance as UX — Core Web Vitals [HARD]

| Metric | Good | Needs work | Poor |
|---|---|---|---|
| LCP | ≤2.5s | 2.5–4s | >4s |
| INP | ≤200ms | 200–500ms | >500ms |
| CLS | ≤0.1 | 0.1–0.25 | >0.25 |

Measured at the 75th percentile. CLS is a design failure as much as an engineering one — reserve space for images, embeds, and async content in the spec.

## Establishing a system from scratch

Decision order: type → spacing → color → radius/shadow/motion.

**Color ramps** — don't step HSL lightness evenly (perceived lightness diverges badly across hues). Use a perceptually uniform space: OKLCH (step L evenly, hold C and H) or Material's HCT tonal palettes.

**Radix Colors' 12-step scale** — the single best shortcut, each step has an assigned role:

| Steps | Role |
|---|---|
| 1–2 | App and subtle backgrounds |
| 3–5 | Component backgrounds: normal, hover, active |
| 6–8 | Borders: subtle, standard, strong / focus ring |
| 9–10 | Solid fills (9 is the pure brand color), hover |
| 11–12 | Text: low-contrast and high-contrast |

Steps 11/12 are built to hit APCA Lc 60 and Lc 90 against step 2 — adopting this scale turns color decisions into lookups instead of judgment calls.

**Neutrals** — tint slightly toward the brand hue (or deliberately warm/cool); pure gray reads cold and clinical.

**Dark mode is not an inversion** [CONVENTION, strongly held] — build a separate ramp. Elevation is expressed by lighter surfaces, not shadows; saturated colors need desaturating to avoid vibration against dark backgrounds.

**Minimum viable token set:** ~15 colors (primary, secondary, 4 semantic, 6–10 neutrals) · 8 spacing steps · 4–5 type sizes, 2–3 weights, 2–3 line heights · 4 radii · 3 shadows · 2–3 motion durations + easings · 3 breakpoints · 5–7 z-layers.

Worth adopting wholesale: Radix Colors, Open Props, Tailwind's default theme, USWDS or GOV.UK (accessibility-first, government-tested).

## Heuristics and laws — vocabulary for a critique

Naming the violated principle makes a critique reviewable instead of a matter of taste.

- **Nielsen's 10 usability heuristics** — visibility of system status · match to the real world · user control and freedom · consistency and standards · error prevention · recognition over recall · flexibility and efficiency · aesthetic and minimalist design · help users recover from errors · help and documentation.
- **Shneiderman's 8 golden rules** — consistency · universal usability · informative feedback · dialogs that yield closure · error prevention · easy reversal · user control · reduced short-term memory load.
- **Gestalt** — proximity · similarity · closure · continuity · figure/ground · common region · common fate · uniform connectedness · Prägnanz. Behind most spacing/grouping critique: "these two cards read as one group because proximity overrides the border" is precise and checkable.
- **Norman** — affordances · signifiers · constraints · mappings · feedback · conceptual model. Best for diagnosing *why* something is confusing.
- **Fitts's law** — acquisition time grows with distance, shrinks with target size; screen edges/corners are effectively infinite targets.
- **Hick's law** — decision time grows logarithmically with option count; chunking reduces effective n. Doubling options does not double decision time.
- **Jakob's law** — users bring expectations from other products; deviate deliberately, not accidentally.
- **Doherty threshold** — ~400ms; see Motion above.
- **Von Restorff effect** — the item that differs is remembered; the mechanism behind a single accent color.
- **Serial position effect** — first and last items are recalled best; order menus accordingly.
- **Peak–end rule** — experiences are remembered by their peak and their ending, not their average.
- **Aesthetic–usability effect** (Kurosu & Kashimura, 1995) — attractive interfaces are *perceived* as more usable and buy tolerance for minor flaws; it doesn't fix real problems.
- **Goal-gradient effect** — effort increases near a visible goal; basis for progress indicators.
- **Tesler's law** — complexity is conserved, moves between user and implementation, doesn't vanish.

**Handle with care** — see [reference-claims.md](reference-claims.md) before citing: Miller's 7±2, Zeigarnik effect, Postel's law.
