# Site images

`images/sources/` contains local masters. `images/catalog.json` is the source of truth for their stable IDs, filenames, rendition presets, and provenance where known. An empty `presets` list archives a master without publishing it. The running Go app embeds only generated renditions, not the masters.

## Replace or add a photo

1. To replace a photo in the same role, keep its ID and replace the master. Update `source` if its format changes, remove the superseded master, and record updated provenance. For a new placement, add a short lowercase ID, a master with its actual format extension, and the presets the placement uses. Use a local file; URLs in the catalog are credits, not inputs to download.
2. Reference the ID, not a public path, from the page template, `main.go`, or `content/{staff,services}.json`. Copy the nearest existing placement, including its `sizes` and loading behavior. Put descriptive alt text in the content or template; use empty alt text for purely decorative images.
3. Run `./scripts/images generate` from the checkout with Docker running. Review and include the source, catalog, and changes under `static/assets/images/` in the same patch. Do not edit generated files or the manifest by hand.
4. Run `./scripts/images check` and `go test ./...`. Open the affected page at mobile and desktop widths. Confirm the crop, transparency, and chosen `currentSrc`; for staff photos, confirm the large rendition is requested only after opening the bio.

Checkouts already contain the generated images. Neither `./dev.sh` nor the application container needs Pillow or Docker-based generation to serve them. Kamal's `pre-build` hook rejects stale outputs before a deployment build.

## Renditions and placement

| Preset | Use |
| --- | --- |
| `hero` | Main page photo. Keep it eager and give it high fetch priority. |
| `inline` | Story, quote, and other content photos. Lazy-load those below the fold. |
| `thumbnail` | Staff and Instagram cards. Use the width of a card, not the entire grid, in `sizes`. |
| `portrait` | Larger staff bio photos; assigned by the existing dialog-open handler. |
| `detail` | Small service illustrations and photos. |
| `brand` | Responsive shared logo. |
| `icon` | PNG favicon; no `srcset` needed. |
| `vector` | Trusted SVG kept as vector, not rasterized. |

Preset widths and encoding settings live in `images/generate.py`. Raster inputs are decoded based on their actual format; the generator keeps aspect ratio and alpha, never upscales, and removes metadata after applying orientation. CSS controls the displayed crop. The generated manifest records each rendition's filename and intrinsic dimensions; the shared image rendering code uses it to avoid hand-entered dimensions. `sizes` must match the placement's CSS breakpoints. Production rendition URLs use the same content fingerprint and immutable cache policy as CSS and JavaScript; local development uses plain URLs.

## Reproducibility and troubleshooting

The wrapper uses Linux/amd64 on all hosts, a digest-pinned Python base, and a hash-pinned Pillow wheel (`images/toolchain/`). Docker can fetch these dependencies on the first build; generation and checking run with networking disabled. On ARM hosts, Docker must support running amd64 images. A second `generate` with unchanged inputs leaves the generated files untouched. `check` uses the same generator with a read-only checkout and reports missing, stale, or extra output. If a photo is wrong, fix the master, preset, or placement and regenerate; never patch a rendition. To upgrade the encoder, update the pins together, regenerate all outputs, and review the byte and visual changes before committing.

For performance work, compare the selected resource in the browser Network panel at the same viewport and DPR. Use a cache-disabled, throttled run for LCP comparisons and report local lab numbers separately from production measurements.
