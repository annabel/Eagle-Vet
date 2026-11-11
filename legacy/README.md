# Eagle Veterinary Hospital

A custom homepage redesign for Eagle Veterinary Hospital in Eagle, Idaho. Built with HTML, CSS, and JavaScript, with a warm palette, playful shapes, the clinic’s existing logo, and photography from its current website.

**[View the demo](https://annabel.github.io/Eagle-Vet/)** · [Current clinic website](https://www.eaglevethosp.com/)

## Project status

This is a published homepage prototype. Appointment requests, service details, team profiles, the client portal, and ordering links lead to the existing clinic website. The demo does not collect form submissions or patient information. Search indexing is disabled, and no analytics tags have been added.

The hero uses the original homepage’s photograph of two dogs running, with the headline “Your pet’s health is our top priority.” The log cabin appears in the clinic story section.

## Run locally

Requires Node.js 18 or newer. No dependency installation or build step is needed.

```bash
npm run dev
```

Open the URL printed in the terminal (normally [http://127.0.0.1:4173/](http://127.0.0.1:4173/)). If the port is already in use, the server automatically tries the next available port, up to 10 ports in total. Refresh the browser after editing files.

To use a different port:

```bash
PORT=4174 npm run dev
```

## Project files

| File | Purpose |
| --- | --- |
| `index.html` | Homepage structure, copy, navigation, and links |
| `styles.css` | Responsive layouts, colors, typography, and decorative shapes |
| `content.json` | Service titles, descriptions, icons, and destination anchors |
| `app.js` | Service rendering, mobile navigation, and footer year |
| `assets/` | Logo, photographs, and font stylesheet |
| `server.mjs` | Local preview server; not needed by GitHub Pages |
| `.nojekyll` | Serves the static site without Jekyll processing |

Typography uses DM Sans and Bricolage Grotesque. The local font stylesheet references Google-hosted font files.

## Publish updates

GitHub Pages publishes the root of the `main` branch. Its automatic deployment runs after changes are pushed; no custom workflow or build pipeline is required.

```bash
git add index.html styles.css content.json app.js assets/
git commit -m "Update homepage"
git push origin main
```

Stage whichever files you actually changed, including `README.md` for documentation updates.

- **Repository:** [annabel/Eagle-Vet](https://github.com/annabel/Eagle-Vet)
- **SSH origin:** `git@github.com:annabel/eagle-vet.git`
- **Deployment status:** [GitHub Actions](https://github.com/annabel/Eagle-Vet/actions)
- **Publishing settings:** Settings → Pages → Deploy from a branch → `main` → `/ (root)`

Keep asset and content paths relative so the site works under `/Eagle-Vet/` as well as locally.

## Future content editing

Content currently lives in code and local files. Service records are already separated into `content.json`; other editable copy remains in `index.html`. If the clinic needs its own editor, move staff, hours, specials, and other copy into structured files and add a lightweight editing layer such as Pages CMS.

## Photography and launch notes

- `assets/logo-mark.png` is the existing illustrated clinic logo.
- `assets/original-home-hero.jpg` is the original homepage photograph (`mia-anderson-2k6v10Y2dIg-unsplash.jpg`).
- “Fido & Max, Eagle, ID” is a placeholder caption requested for the demo; the dogs’ names and location are unverified.
- `assets/cabin-reference.png` is the supplied exterior screenshot. The displayed `assets/cabin-demo.png` is an AI-retouched version with the map pins removed. Replace it with an original exterior photograph before the final launch.
- The cabin edit used the built-in image generation tool with the instruction to remove only the two map pins and halos while preserving the building, sign, landscaping, framing, and photographic character. Generated edits can alter details.
- `assets/pets-hero.png` is an unused generated concept image. The displayed animal photograph is from the original website.
- The testimonial excerpt is attributed to Robyn Boersma on the existing homepage.
- Opening times conflict on the existing website. The demo lists weekdays and links to current hours until the owner confirms the schedule.

Before replacing the clinic’s live site, confirm copy, photography rights, captions, hours, and destination links with the owner, and plan redirects for any changed URLs.
