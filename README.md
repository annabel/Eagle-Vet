# Eagle Veterinary Hospital

A hand-written Go website for Eagle Veterinary Hospital in Eagle, Idaho. Shared server-rendered HTML templates, plain CSS, and a little vanilla JavaScript. No web framework, CMS, database, frontend build tool, or third-party Go modules.

## Run locally

Requires Go 1.22 or newer. From this directory:

```sh
./dev.sh
```

Or, with Go on your PATH:

```sh
go run .
```

Open the URL printed in the terminal. The server starts at **http://127.0.0.1:4173** and tries the next port when a port is occupied. Choose another starting port with `PORT=4180 ./dev.sh`.

The launcher also recognizes a local, ignored `.tools/go` toolchain. A checksum-verified official toolchain was placed there for development on this machine; it is not part of the source or a dependency to commit. Install Go normally on another machine.

Templates, assets, and content are embedded in the executable. Stop and restart after edits to see changes. `npm run dev` is retained only as a compatibility shortcut to `./dev.sh`; Node is not required when using Go directly.

## Build and check

```sh
go test ./...
go build -o bin/eagle-vet .
./bin/eagle-vet
```

The single binary contains the templates and assets. It currently binds to localhost for development. Production hosting, public binding, TLS, and form delivery are not configured.

## Structure

| Path | Purpose |
| --- | --- |
| `main.go` | Standard-library HTTP server, route handling, and template rendering |
| `main_test.go` | Route, local-link, anchor, asset, and demo-submission checks |
| `templates/layout.html` | Shared document structure |
| `templates/partials/` | Header, navigation, footer, page hero, and contact form |
| `templates/pages/` | Individual pages, including the approved homepage |
| `content/` | Hard-coded staff bios, navigation, services, testimonials, offers, and gallery records |
| `static/` | CSS, JavaScript, logo, and existing-site photography |
| `docs/content-review.md` | Items to confirm with the clinic before launch |
| `docs/image-sources.json` | URLs for photographs downloaded from the current clinic site |
| `legacy/` | Backup of the previous static homepage and Node preview, including its local port fix |

Most page copy lives directly in the templates. Shared records live in JSON files for easy maintenance. `html/template` escapes content when rendering. There is no content editor.

## Navigation and pages

The current site’s navigation groups and destination anchors are preserved:

- **Home** → `/`
- **About Us** → Meet The Team (`/meettheteam`), Testimonials. The `/meettheteam` page starts with the team; the former overview link and introductory sections have been removed.
- **What We Do** → Services and its eight sections: wellness, injury, dentistry, surgery, urgent care, exotics, naturopathic care, and pharmacy
- **Resources** → Patient Resources, Client Portal, Online Ordering, Client Specials
- **Visit Us** → `/visit-us`

Also included: `/appointment`, `/holiday-hours`, `/photo-gallery`, `/hello`, and a designed 404 page. `/home`, `/about`, `/about-us`, `/what-we-do`, and `/resources` redirect to their corresponding content pages. Trailing slashes redirect to canonical paths.

## Forms and integrations

**Contact and appointment forms are local demonstrations.** Their buttons validate the fields and show a message explicitly stating that nothing was sent or saved. There is no email delivery, storage, booking, analytics, or successful appointment confirmation. POST requests are rejected. Input values are not included in URLs or browser storage. With JavaScript disabled, the preview buttons remain disabled.

The client portal preserves the existing IDEXX Neo configuration (company `8597`, branch `1`), loading its script only when requested. Login and account behavior belong to IDEXX Neo; authenticated portal behavior has not been tested. Vetsource and Purina ordering links retain the current clinic destinations. Phone, email, directions, review, and social links use the current clinic information.

Search indexing is disabled through robots metadata, response headers, and `robots.txt` while this remains a preview.

## Design and content

The approved homepage palette, rounded typography, arches, doodles, logo, two-dog hero photo, headline, and caption are carried through the new pages. Existing clinic photos and staff bios were reused from the current website on October 30, 2026. No new animal imagery was generated for this rebuild.

The hero caption “Fido & Max, Eagle, ID” remains an unverified placeholder. The cabin image remains the previously retouched screenshot; use an original exterior photo before launch. The unused generated animal image remains only in the legacy assets, not in the active app.

Hours conflict and holiday dates are stale on the current site. The redesign shows weekdays, asks visitors to confirm opening times, and avoids presenting old holiday dates as current. Wellness-profile prices mirror the source and require owner confirmation. See `docs/content-review.md`.

The homepage features one Instagram post. To update it, replace the image, date, caption, and post link in `templates/pages/home.html` and record the source in `docs/image-sources.json`. There is no live feed; replace or remove dated promotions after they end.

## Publishing

This Go app requires a host that runs a Go executable. **GitHub Pages cannot run it.** The existing published static demo is unchanged. No hosting settings, GitHub workflows, commits, or pushes were made during this rebuild. The original static files are backed up in `legacy/` for reference.

All changes are intentionally local and uncommitted. Commit and publish only after your own review.
