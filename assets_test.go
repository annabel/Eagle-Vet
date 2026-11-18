package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"testing"
)

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
	for _, original := range []string{"/static/site.css", "/static/pages.css", "/static/site.js", "/assets/fonts.css"} {
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
	for _, url := range []string{"/static/site.css", "/static/pages.css", "/static/site.js", "/assets/fonts.css"} {
		if !strings.Contains(home.Body.String(), `"`+url+`"`) {
			t.Errorf("development HTML missing plain asset URL %s", url)
		}
	}
}
