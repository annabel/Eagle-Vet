package main

import (
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func testApp(t *testing.T) *application {
	t.Helper()
	a, err := newApplication()
	if err != nil {
		t.Fatal(err)
	}
	return a
}
func get(a *application, target string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodGet, target, nil)
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	return w
}
func TestPagesAndLocalLinks(t *testing.T) {
	a := testApp(t)
	hrefs := regexp.MustCompile(`(?:href|src)="([^"]+)"`)
	for route := range a.pages {
		t.Run(route, func(t *testing.T) {
			w := get(a, route)
			if w.Code != 200 {
				t.Fatalf("GET %s: %d", route, w.Code)
			}
			if !strings.Contains(w.Body.String(), `aria-label="Main navigation"`) {
				t.Fatal("Shared navigation missing")
			}
			for _, match := range hrefs.FindAllStringSubmatch(w.Body.String(), -1) {
				raw := html.UnescapeString(match[1])
				u, err := url.Parse(raw)
				if err != nil {
					t.Fatal(err)
				}
				if u.IsAbs() || u.Host != "" {
					continue
				}
				target := u.Path
				if target == "" {
					target = route
				}
				response := get(a, target)
				if response.Code != 200 {
					t.Errorf("Broken local destination %s on %s: %d", raw, route, response.Code)
				}
				if u.Fragment != "" && !strings.Contains(response.Body.String(), `id="`+u.Fragment+`"`) {
					t.Errorf("Missing anchor %s on %s", raw, route)
				}
			}
		})
	}
}
func TestPreviewRejectsSubmissions(t *testing.T) {
	a := testApp(t)
	for _, route := range []string{"/visit-us", "/appointment"} {
		w := httptest.NewRecorder()
		a.ServeHTTP(w, httptest.NewRequest("POST", route, strings.NewReader("name=Demo")))
		if w.Code != 405 {
			t.Errorf("%s accepted a submission: %d", route, w.Code)
		}
		if w.Header().Get("Allow") != "GET, HEAD" {
			t.Error("Missing allowed methods")
		}
	}
}
func TestAliasesAndNotFound(t *testing.T) {
	a := testApp(t)
	for from, to := range map[string]string{"/home": "/", "/about-us": "/meettheteam", "/about": "/meettheteam", "/what-we-do": "/services", "/resources": "/patient-resources", "/about/": "/meettheteam", "/meettheteam/": "/meettheteam"} {
		w := get(a, from)
		if w.Code != 301 || w.Header().Get("Location") != to {
			t.Errorf("%s redirect: %d %s", from, w.Code, w.Header().Get("Location"))
		}
	}
	for _, route := range []string{"/missing", "/assets/missing.jpg", "/assets/../main.go", "/.git/config", "/content/staff.json", "/templates/layout.html"} {
		if w := get(a, route); w.Code != 404 {
			t.Errorf("%s unexpectedly exposed: %d", route, w.Code)
		}
	}
}
func TestHeadAndRobots(t *testing.T) {
	a := testApp(t)
	w := httptest.NewRecorder()
	a.ServeHTTP(w, httptest.NewRequest("HEAD", "/meettheteam", nil))
	if w.Code != 200 || w.Body.Len() != 0 {
		t.Fatal("HEAD must return headers without a body")
	}
	if w.Header().Get("X-Robots-Tag") != "noindex, nofollow" {
		t.Error("Preview indexing header missing")
	}
	if w := get(a, "/robots.txt"); !strings.Contains(w.Body.String(), "Disallow: /") {
		t.Error("Preview robots rule missing")
	}
}
