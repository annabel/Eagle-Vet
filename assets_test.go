package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"path"
	"regexp"
	"strings"
	"testing"
)

var sharedAssetURLs = []string{
	"/static/site.css",
	"/static/pages.css",
	"/static/site.js",
	"/assets/fonts/bricolage-grotesque-latin.woff2",
	"/assets/fonts/dm-sans-latin.woff2",
}

func TestFingerprintedAssetsAndCaching(t *testing.T) {
	a, err := newApplication(true)
	if err != nil {
		t.Fatal(err)
	}
	home := get(a, "/")
	missing := get(a, "/missing")
	for _, page := range []*httptest.ResponseRecorder{home, missing} {
		if got := page.Header().Get("Cache-Control"); got != "no-cache" {
			t.Errorf("HTML status %d: Cache-Control = %q, want no-cache", page.Code, got)
		}
	}
	for _, original := range sharedAssetURLs {
		t.Run(original, func(t *testing.T) {
			plain := get(a, original)
			if plain.Code != http.StatusOK || strings.Contains(plain.Header().Get("Cache-Control"), "immutable") {
				t.Fatalf("unversioned asset: status %d, cache %q", plain.Code, plain.Header().Get("Cache-Control"))
			}
			sum := sha256.Sum256(plain.Body.Bytes())
			ext := path.Ext(original)
			versioned := fmt.Sprintf("%s.%x%s", strings.TrimSuffix(original, ext), sum[:16], ext)
			for _, page := range []*httptest.ResponseRecorder{home, missing} {
				if !strings.Contains(page.Body.String(), `"`+versioned+`"`) {
					t.Errorf("HTML status %d missing asset URL %s", page.Code, versioned)
				}
			}
			asset := get(a, versioned)
			if asset.Code != http.StatusOK || !bytes.Equal(asset.Body.Bytes(), plain.Body.Bytes()) {
				t.Fatalf("fingerprinted URL %s: status %d or content mismatch", versioned, asset.Code)
			}
			if got := asset.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
				t.Errorf("fingerprinted URL Cache-Control = %q", got)
			}
			if ext == ".woff2" {
				if got := asset.Header().Get("Content-Type"); got != "font/woff2" || !bytes.HasPrefix(asset.Body.Bytes(), []byte("wOF2")) {
					t.Errorf("fingerprinted font %s: expected WOFF2, Content-Type = %q", versioned, got)
				}
			}
			unknown := get(a, strings.TrimSuffix(original, ext)+"."+strings.Repeat("0", 32)+ext)
			if unknown.Code != http.StatusNotFound || strings.Contains(unknown.Header().Get("Cache-Control"), "immutable") {
				t.Errorf("unknown fingerprint: status %d, cache %q", unknown.Code, unknown.Header().Get("Cache-Control"))
			}
		})
	}
}

func TestDevelopmentUsesPlainAssetURLs(t *testing.T) {
	a := testApp(t)
	home := get(a, "/")
	for _, url := range sharedAssetURLs {
		if !strings.Contains(home.Body.String(), `"`+url+`"`) {
			t.Errorf("development HTML missing plain asset URL %s", url)
		}
	}
}

func TestResponsiveImagesAreServed(t *testing.T) {
	a, err := newApplication(true)
	if err != nil {
		t.Fatal(err)
	}

	tags := regexp.MustCompile(`<(?:img\b|link rel="icon")[^>]*>`)
	attrs := regexp.MustCompile(` (src|srcset|data-src|data-srcset|href)="([^"]+)"`)
	width := regexp.MustCompile(`^[1-9][0-9]*w$`)
	mimeTypes := map[string]string{".webp": "image/webp", ".png": "image/png", ".svg": "image/svg+xml"}
	seen := make(map[string]bool)
	check := func(route, asset string) {
		t.Helper()
		if seen[asset] {
			return
		}
		seen[asset] = true
		if !strings.HasPrefix(asset, "/assets/images/") {
			t.Errorf("%s: %s bypasses generated image assets", route, asset)
			return
		}
		res := get(a, asset)
		if res.Code != http.StatusOK || res.Header().Get("Cache-Control") != "public, max-age=31536000, immutable" {
			t.Errorf("%s: %s returned status %d and cache %q", route, asset, res.Code, res.Header().Get("Cache-Control"))
			return
		}
		want := mimeTypes[path.Ext(asset)]
		if got, _, _ := strings.Cut(res.Header().Get("Content-Type"), ";"); want == "" || got != want {
			t.Errorf("%s: %s Content-Type = %q, want %q", route, asset, got, want)
		}
		if want == "image/webp" || want == "image/png" {
			if got := http.DetectContentType(res.Body.Bytes()); got != want {
				t.Errorf("%s: %s body format = %q, want %q", route, asset, got, want)
			}
		}
	}

	routes := []string{"/missing"}
	for route := range a.pages {
		routes = append(routes, route)
	}
	for _, route := range routes {
		res := get(a, route)
		want := http.StatusOK
		if route == "/missing" {
			want = http.StatusNotFound
		}
		if res.Code != want {
			t.Errorf("GET %s: status %d, want %d", route, res.Code, want)
			continue
		}
		for _, tag := range tags.FindAllString(res.Body.String(), -1) {
			for _, attr := range attrs.FindAllStringSubmatch(tag, -1) {
				value := html.UnescapeString(attr[2])
				if !strings.HasSuffix(attr[1], "srcset") {
					check(route, value)
					continue
				}
				for _, candidate := range strings.Split(value, ",") {
					fields := strings.Fields(candidate)
					if len(fields) != 2 || !width.MatchString(fields[1]) {
						t.Errorf("%s: invalid image candidate %q", route, candidate)
						continue
					}
					check(route, fields[0])
				}
			}
		}
	}
}
