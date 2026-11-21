# Eagle Vet contributor guide

- Run the site with `./dev.sh`; run Go checks with `go test ./...`.
- For any image change, follow [docs/images.md](docs/images.md). Edit the local master and `images/catalog.json`, not `static/assets/images/`. Use logical image IDs in page records, content JSON, and templates; never hard-code generated filenames or cache hashes.
- Run `./scripts/images generate` after changing image inputs or presets. Include the masters, catalog, and generated changes in the same patch. Use `./scripts/images check` to verify them without writing files.
- Keep alt text, responsive `sizes`, and loading priority at the placement. Check the affected page at mobile and desktop widths. Do not add a runtime image encoder, fetch remote masters during generation, or modify `legacy/` to update the current site.
