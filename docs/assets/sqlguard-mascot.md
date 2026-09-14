# Release Artwork Provenance

## Asset

- File: [`sqlguard-mascot.png`](sqlguard-mascot.png)
- Intended use: compact mascot illustration in the repository README
- Dimensions: 600 × 600 pixels
- SHA-256: `79ebaba75027de00f829bcc94b7f9179af270c0efcb9a566562ac0eea4afd6a8`
- README display width: 240 pixels
- README alt text: “A turquoise gopher holding a sword protectively in front
  of a friendly blue elephant-shaped database”

## Generation

- Date: 2026-09-14
- Tool: OpenAI built-in image generation tool in Codex
- Mode: new image generation; no input or reference images were supplied to
  the generator
- Use case: `stylized-concept`

Before generation, the maintainer-facing visual review used the
[`pressly/goose` mascot](https://github.com/pressly/goose/blob/main/assets/goose_logo.png)
only to identify broad traits such as compact composition, flat color, and bold
hand-drawn outlines. The source image was not passed to the generator, and the
prompt explicitly prohibited reproducing its mascot, costume, pose, or
composition.

Prompt:

```text
Use case: stylized-concept
Asset type: small README mascot illustration preview
Primary request: Create a compact original mascot illustration of a Go-gopher-inspired character holding a small sword protectively in front of a friendly blue elephant character that evokes a PostgreSQL database without reproducing the official PostgreSQL Slonik logo.
Scene/backdrop: genuinely transparent background, no scene and no border.
Subject: one turquoise gopher in front, alert but friendly, holding a short sword sideways as a protective barrier; one friendly blue elephant slightly behind the gopher, with a recognizable elephant head and trunk plus a subtle database-cylinder body cue. The elephant should feel familiar to PostgreSQL users through its blue color and database context, but have a distinct full-body silhouette, face, proportions, ears, and pose unlike Slonik.
Style/medium: playful hand-drawn open-source mascot sticker; simple flat fills, bold slightly irregular deep-purple outlines, minimal shading, charming imperfect linework, strong readable silhouettes. Use only the broad visual language of compact hand-drawn mascot art; do not reproduce the pressly/goose mascot, costume, pose, or composition.
Composition/framing: small nearly square icon, centered pair, generous transparent padding, readable at 180–240 px; gopher clearly shielding rather than attacking the elephant.
Color palette: bright turquoise, PostgreSQL-adjacent blue, white accents, deep purple outlines, small warm orange or yellow detail on the sword.
Text: none.
Constraints: original artwork only; no exact PostgreSQL logo, no Slonik tracing or modification, no PostgreSQL wordmark, no Go logo, no goose costume or goose character, no brand lockup, no sponsorship or endorsement cue, no blood or violence, no extra characters, no letters, no watermark. The sword is symbolic and protective.
Avoid: 3D rendering, cinematic lighting, realistic fur, detailed background, gradients, glossy surfaces, aggressive expressions, exact imitation of any existing mascot artwork.
```

## Post-processing

The generator returned a 1254 × 1254 RGBA PNG. It was resampled to 600 × 600
pixels with macOS `sips`, preserving its transparent background and PNG format.
There was no crop, compositing, inpainting, or manual semantic edit. The
optimized file is about 253 KiB; the transient source PNG is not distributed
with the repository.

## Review

The asset was reviewed at its generated dimensions and at a 240 × 240 README
preview. The maintainer then selected it to replace the initial wide artwork.
At README size:

- the gopher, sword, elephant, and database-cylinder cue remain recognizable;
- the compact foreground/background arrangement reads as the gopher guarding
  the elephant/database character;
- the transparent background integrates with both light and dark README themes;
- there is no visible text, signature, watermark, wordmark, or official logo;
- the full-body elephant/database character has a distinct silhouette and does
  not reproduce, trace, modify, or combine the official Slonik mark; and
- the composition contains no claim or visual cue of PostgreSQL project
  sponsorship, affiliation, or endorsement.

## Attribution and trademarks

The Go gopher was designed by Renée French. This generated illustration uses
an adapted Go-gopher-inspired character and credits the original design under
the [Creative Commons Attribution 4.0 International license (CC BY 4.0)](https://creativecommons.org/licenses/by/4.0/).
The Go project's [FAQ](https://go.dev/doc/faq#go_or_golang) documents the
creator and license.

The blue elephant/database character is original to this generated composition
and is not the PostgreSQL Elephant Logo (Slonik). PostgreSQL and Slonik are
trademarks of the PostgreSQL Community Association of Canada; see the
[PostgreSQL Trademark Policy](https://www.postgresql.org/about/policies/trademarks/).
No-affiliation statement: this project and artwork are not affiliated with,
sponsored by, endorsed by, or approved by the PostgreSQL Project or its
trademark owner.
