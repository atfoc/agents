# UX claims ledger — check before citing a number

This is defensive. Every item below circulates widely in UX writing and failed verification. Repeating one produces a confident, checkable-and-wrong rationale — worse than giving no rationale at all. Check any statistic, percentage, or memory/attention "limit" against this list before stating it as fact in a spec or critique (SKILL.md Step 11).

| Claim | Verdict | What's actually true |
|---|---|---|
| "Cramped screens → users abandon 47% faster" | **Fabricated** | Traces to Gloria Mark's finding that average attention *per screen* is ~47 **seconds**. A duration was converted into a fake percentage. |
| "Generous spacing boosts conversions 28%" | **Unsupported** | No traceable study. Whitespace does help; documented effects range ~20–35% depending on metric and context. Don't quote a single figure. |
| "Pure black on white cuts reading speed 20%" | **Backwards** | Piepenbrock, Mayr & Buchner (2014): positive polarity (black on white) produced *better* proofreading speed and accuracy. The ~20% figure describes *reduced* contrast. Keep the near-black convention — justify it by long-session visual fatigue and refinement, not legibility. |
| "Recognition 85–95% vs recall 35–50%" | **Directionally right, falsely precise** | Recognition reliably beats recall (Yonelinas; Karpicke & Roediger 2008; Tulving & Pearlstone). The exact percentages are an amalgam, not one study. State the direction, not the digits. |
| "Max 7 menu items (Miller's law)" | **Contradicted** | Miller (1956) concerned absolute judgment of unidimensional stimuli and immediate serial recall — not visible interface options; he called the recurring seven "a pernicious, Pythagorean coincidence." Cowan (2001) puts working memory near 4 chunks. Visible menus are a *recognition* task, not a memory task — Landauer & Nachbar (1985) found broader, shallower menus outperform deeper ones; Kiger (1984) suggested 8–9 per level. A 7-item cap forces deeper hierarchies and makes things worse. |
| "Never nest deeper than 3 levels" | **Partly right, wrong shape** | Depth does degrade performance, but gradually — no validated cliff at 3. Defensible version: prefer breadth over depth; 2–3 levels ideal; watch closely past 4, especially on mobile. |
| "Decorative images get zero eye fixations" | **Overstated** | NNGroup describes the behavior categorically — users ignore decorative images and concentrate fixations on text — without quantifying it as zero. Say "substantially fewer," not "zero." |

**Handle with care, not in the table above:** Zeigarnik effect (2025 meta-analyses find no reliable recall advantage; the related Ovsiankina resumption effect holds up better) · Postel's law (sound for form input, contested as a protocol rule for security reasons).

**The transferable lesson:** design writing recirculates invented precision. A number with no traceable study is a red flag — oddly specific numbers (47%, 28%) are the most suspicious of all. Prefer a mechanism you can explain over a statistic you can't source.
