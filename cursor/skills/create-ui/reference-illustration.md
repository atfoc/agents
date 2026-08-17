# Imagery and icon reference

Read this when Step 9 of `SKILL.md` needs the full style vocabulary. The output of this reference is always a **written description**, never a generated image or a call to an image tool.

## When an image earns its place

NNGroup eye-tracking finds users attend closely to task-relevant images and quickly learn to ignore decorative ones, spending their fixations on text instead — say "substantially fewer fixations," not "zero" (see [reference-claims.md](reference-claims.md)).

Earns it: empty states · onboarding · error pages · explaining an abstract process · brand personality in marketing · wayfinding.
Doesn't: generic decoration next to text that already explains itself · anything competing with the primary action · imagery that breaks in dark mode · mixed styles across one product.

## The style specification vocabulary

A style must be *described*, not gestured at. Cover each axis for every image:

- **Line** — stroke present or not; weight; cap and join; hard vs. textured edge
- **Shape** — geometric vs. organic; grid-constructed vs. freehand
- **Fill** — palette size; relationship to brand color; flat vs. gradient
- **Detail level** — readable at 16px / 32px / 128px+
- **Perspective** — flat, isometric (30°), 2.5D, linear perspective, orthographic
- **Texture** — clean vector, grain, halftone, painterly
- **Lighting** — direction, shadow type, highlights present or not
- **Character** — faceless/simplified/detailed; proportions; expression range; diversity
- **Composition** — centering, negative space ratio, background treatment, crop

## Set coherence

Across a set of images, these axes must hold or the set reads as clip art: line weight · corner radius · perspective · palette · detail level · character proportions · prop scale · background treatment · texture presence · shape language. The cheapest guarantee of coherence is **one source** — describe every image in the set against the same style reference or the same fixed style block.

## Named styles

- **Flat** — single plane, 2–5 solid colors, no depth. Scales, recolors, loads well; safe but generic.
- **Corporate Memphis / Alegria** — flat with exaggerated limbs, small heads, faceless figures, non-representational skin tones, heavily oversaturated. Widely criticized now; signals "default" more than "friendly."
- **Isometric technical** — 30° axes, true scale, no vanishing point. Best for explaining systems and processes.
- **Hand-drawn / textured** — visible stroke, organic line, grain. Warm and distinctive; hardest to keep consistent across a set.
- **3D / claymorphism** — soft-body forms, pastel palettes, ambient light. Playful; showing fatigue as a trend.
- **Line art** — stroke-only, consistent weight. The dominant modern UI idiom.
- **Spot illustration** — small, single-idea, 100–300px, high contrast for small-scale legibility.

## Icons

Icons are not small illustrations — they obey a grid system: typically 24×24 with a ~20×20 live area, keyline shapes (square ≈18, circle ≈20) that equalize *optical* size across differently-shaped glyphs, and one stroke weight (commonly 2px) held across the whole set.

| Library | Grid | Weight | Character |
|---|---|---|---|
| Lucide / Feather | 24 | 2px | Minimal, slightly rounded; the modern default |
| Heroicons | 24 (+ sizes) | varies | Outline/solid/mini; Tailwind-native |
| Phosphor | 24 | 6 weights | Thin→Bold, Fill, Duotone; most flexible |
| Material Symbols | variable | variable | Outlined/Rounded/Sharp; variable font |
| Radix | 15 | light | Tiny, crisp, precise |
| Tabler | 24 | adjustable | Largest free set |

Specify `currentColor` (or equivalent) so icons inherit text color and follow the theme. Icon-only controls need an accessible name in the spec; icon+label is safer for anything not universally understood. Never spec emoji as functional icons — they can't inherit color, won't adapt to theme, and have no states.

## Writing a reproducible description

Anatomy: subject → style descriptors (the axes above) → composition → palette (actual hex values, not "brand colors") → technical specs (aspect ratio, background/transparency) → negative constraints.

Consistency across a set, in order of effectiveness: a style-reference image the generator can be pointed at · a fixed style block repeated verbatim across every description, varying only the subject line · seed control where the target tool supports it · prompt discipline — short and non-contradictory beats long and hedged.

Known failure modes worth writing a negative constraint against: character drift across a set · garbled text inside images (never ask for readable text in-image — composite it afterward) · uncanny proportions · palette drift · unusable/cluttered backgrounds · wrong aspect ratio · style mixing within one image.

Licensing terms differ by model, tier, and training-data provenance, and are contested — for anything commercial, tell the user to verify rather than assuming.

**Open sets worth knowing, if a stock option beats a generated one:** unDraw (CC0, recolorable), Humaaans (CC0, composable), Open Peeps (CC0), DrawKit (MIT tier).

## Delivery format (for the spec, not for generation)

Prefer SVG for icons, diagrams, and geometric illustration — it scales, themes via `currentColor`, animates, stays small. Prefer raster (WebP with PNG fallback) for photographic or highly detailed organic work, where SVG path count explodes. Always spec a `viewBox` and let CSS control size; for dark mode, `currentColor`/custom properties beat shipping two files.

**Alt text:** decorative → empty alt, hidden from screen readers. Informative → describe the *information*, not the artwork: "Diagram: three clients connecting to one server," not "isometric illustration in blue and green."
