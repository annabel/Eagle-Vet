package main

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

//go:embed templates static content
var files embed.FS

type Link struct{ Label, URL string }
type NavGroup struct {
	Label, URL string
	Links      []Link
}
type Person struct{ Name, Role, Image, Bio string }
type Service struct {
	ID, Title, Intro, Image, ImageAlt, ImagePreset string
	Items                                         []string
	More                                          []Detail
}
type Detail struct{ Title, Text string }
type Quote struct{ Text, Author string }
type Page struct {
	Path, Name, Title, Heading, Accent, Intro, Image, ImageAlt, Template string
	Nav                                                                  []NavGroup
	Staff                                                                []Person
	Services                                                             []Service
	Quotes                                                               []Quote
	Year                                                                 int
}

type application struct {
	pages            map[string]Page
	templates        map[string]*template.Template
	notFoundTemplate *template.Template
	assetAliases     map[string]string
	static           http.Handler
}

func loadJSON(name string, target any) error {
	b, err := files.ReadFile("content/" + name + ".json")
	if err != nil {
		return err
	}
	return json.Unmarshal(b, target)
}

func newApplication(fingerprintAssets bool) (*application, error) {
	var staff []Person
	var services []Service
	var quotes []Quote
	var nav []NavGroup
	for _, entry := range []struct {
		name   string
		target any
	}{{"staff", &staff}, {"services", &services}, {"quotes", &quotes}, {"navigation", &nav}} {
		if err := loadJSON(entry.name, entry.target); err != nil {
			return nil, fmt.Errorf("content %s: %w", entry.name, err)
		}
	}
	pages := map[string]Page{
		"/":                  {Name: "Home", Title: "Your pet’s health is our top priority", Template: "home"},
		"/meettheteam":       {Name: "Meet the Team", Title: "Meet the Team", Template: "about"},
		"/services":          {Name: "Services", Title: "Our Services", Heading: "A little care goes", Accent: "a long way.", Intro: "From everyday wellness to the unexpected, personalized veterinary care for the Treasure Valley’s dogs, cats, and exotic companions.", Image: "services-hero", ImageAlt: "A cream-colored puppy sitting in the grass", Template: "services"},
		"/testimonials":      {Name: "Testimonials", Title: "Kind Words from Our Clients", Heading: "Happy pets and the", Accent: "people who love them", Intro: "The relationships we build are at the heart of what we do. Here’s what a few of our pet families have shared about their visits.", Image: "testimonials-hero-boat", ImageAlt: "A black-and-white dog sitting on a boat at sunset", Template: "testimonials"},
		"/patient-resources": {Name: "Patient Resources", Title: "Patient Resources", Heading: "Everything you need,", Accent: "a little closer.", Intro: "Find your pet’s client portal and order food and medication online.", Image: "patient-resources-puppy", ImageAlt: "A cream-colored puppy lying down with one paw stretched out", Template: "resources"},
		"/visit-us":          {Name: "Visit Us", Title: "Visit & Contact Us", Heading: "Come visit us at our log cabin.", Intro: "Find us on Palmetto Avenue in Eagle. Whether you’re new here or already part of the family, we’d love to hear from you.", Image: "cabin-demo", ImageAlt: "Eagle Veterinary Hospital’s log cabin on Palmetto Avenue", Template: "visit"},
		"/appointment":       {Name: "Appointment", Title: "Request an Appointment", Template: "appointment"},
		"/hello":             {Name: "Hello", Title: "Welcome to Eagle Vet", Template: "hello"},
	}
	staticFS, err := fs.Sub(files, "static")
	if err != nil {
		return nil, err
	}
	a := &application{pages: pages, templates: map[string]*template.Template{}}
	var assetURLs map[string]string
	if fingerprintAssets {
		assetURLs, a.assetAliases, err = fingerprintedAssets(staticFS)
		if err != nil {
			return nil, err
		}
	}
	assetURL := func(url string) (string, error) {
		if assetURLs == nil {
			return url, nil
		}
		if versioned, ok := assetURLs[url]; ok {
			return versioned, nil
		}
		return "", fmt.Errorf("no fingerprint for asset %s", url)
	}
	images, err := loadImages(staticFS, assetURL)
	if err != nil {
		return nil, fmt.Errorf("image manifest: %w", err)
	}
	image := func(id, preset string) (siteImage, error) {
		if entry, ok := images[id][preset]; ok {
			return entry, nil
		}
		return siteImage{}, fmt.Errorf("no image %q with preset %q", id, preset)
	}
	parsePage := func(name string) (*template.Template, error) {
		return template.New("layout.html").Funcs(template.FuncMap{
			"assetURL": assetURL,
			"image":    image,
		}).ParseFS(files, "templates/layout.html", "templates/partials/*.html", "templates/pages/"+name+".html")
	}
	for route, p := range pages {
		p.Path = route
		p.Nav = nav
		p.Staff = staff
		p.Services = services
		p.Quotes = quotes
		p.Year = time.Now().Year()
		a.pages[route] = p
		t, err := parsePage(p.Template)
		if err != nil {
			return nil, fmt.Errorf("template %s: %w", route, err)
		}
		a.templates[route] = t
	}
	a.notFoundTemplate, err = parsePage("404")
	if err != nil {
		return nil, fmt.Errorf("template 404: %w", err)
	}
	a.static = http.FileServer(http.FS(staticFS))
	return a, nil
}

const imageDir = "assets/images/"

type siteImage struct {
	Src, Srcset   string
	Width, Height int
}

func loadImages(staticFS fs.FS, assetURL func(string) (string, error)) (map[string]map[string]siteImage, error) {
	data, err := fs.ReadFile(staticFS, imageDir+"manifest.json")
	if err != nil {
		return nil, err
	}
	var manifest map[string]map[string][]struct {
		File          string
		Width, Height int
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	// Choose a fallback for a typical 1x slot; browsers with srcset use each placement's sizes.
	preferredWidth := map[string]int{
		"hero": 600, "inline": 400, "thumbnail": 300, "portrait": 260,
		"detail": 132, "brand": 145, "icon": 32, "vector": 132,
	}
	images := make(map[string]map[string]siteImage, len(manifest))
	for id, presets := range manifest {
		images[id] = make(map[string]siteImage, len(presets))
		for preset, candidates := range presets {
			target, known := preferredWidth[preset]
			if !known || len(candidates) == 0 {
				return nil, fmt.Errorf("invalid preset %q for image %q", preset, id)
			}
			var entry siteImage
			var srcset strings.Builder
			previousWidth := 0
			for _, candidate := range candidates {
				if !imageExtension(path.Ext(candidate.File)) ||
					strings.IndexFunc(candidate.File, func(r rune) bool {
						return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' ||
							r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.')
					}) >= 0 || candidate.Width <= previousWidth || candidate.Height <= 0 {
					return nil, fmt.Errorf("invalid candidate %q for image %q (%s)", candidate.File, id, preset)
				}
				previousWidth = candidate.Width
				name := imageDir + candidate.File
				if _, err := fs.Stat(staticFS, name); err != nil {
					return nil, err
				}
				url, err := assetURL("/" + name)
				if err != nil {
					return nil, err
				}
				if entry.Width < target {
					entry.Src, entry.Width, entry.Height = url, candidate.Width, candidate.Height
				}
				if len(candidates) > 1 {
					if srcset.Len() > 0 {
						srcset.WriteString(", ")
					}
					srcset.WriteString(url)
					srcset.WriteByte(' ')
					srcset.WriteString(strconv.Itoa(candidate.Width))
					srcset.WriteByte('w')
				}
			}
			entry.Srcset = srcset.String()
			images[id][preset] = entry
		}
	}
	return images, nil
}

func imageExtension(ext string) bool {
	return ext == ".webp" || ext == ".png" || ext == ".svg"
}

// Fingerprint embedded styles, scripts, fonts, and generated images once, preserving their public URL prefixes.
func fingerprintedAssets(staticFS fs.FS) (map[string]string, map[string]string, error) {
	urls := make(map[string]string)
	aliases := make(map[string]string)
	hash := sha256.New()
	var buffer [32 * 1024]byte
	err := fs.WalkDir(staticFS, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		ext := path.Ext(name)
		if entry.IsDir() || (ext != ".css" && ext != ".js" && ext != ".woff2" &&
			!(strings.HasPrefix(name, imageDir) && imageExtension(ext))) {
			return nil
		}
		file, err := staticFS.Open(name)
		if err != nil {
			return fmt.Errorf("open asset %s: %w", name, err)
		}
		hash.Reset()
		_, err = io.CopyBuffer(hash, file, buffer[:])
		file.Close()
		if err != nil {
			return fmt.Errorf("hash asset %s: %w", name, err)
		}
		sum := hash.Sum(buffer[:0])
		url := "/static/" + name
		if strings.HasPrefix(name, "assets/") {
			url = "/" + name
		}
		versioned := fmt.Sprintf("%s.%x%s", strings.TrimSuffix(url, ext), sum[:16], ext)
		urls[url], aliases[versioned] = versioned, name
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return urls, aliases, nil
}

func (a *application) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "This preview does not accept form submissions.", http.StatusMethodNotAllowed)
		return
	}
	if r.URL.Path == "/robots.txt" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, "User-agent: *\nDisallow: /\n")
		return
	}
	if strings.HasPrefix(r.URL.Path, "/assets/") || strings.HasPrefix(r.URL.Path, "/static/") {
		asset, fingerprinted := a.assetAliases[r.URL.Path]
		if fingerprinted {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			asset = strings.TrimPrefix(r.URL.Path, "/")
			asset = strings.TrimPrefix(asset, "static/")
			if strings.Contains(asset, "..") || strings.Contains(asset, "/.") {
				http.NotFound(w, r)
				return
			}
			if _, err := fs.Stat(files, "static/"+asset); err != nil {
				http.NotFound(w, r)
				return
			}
		}
		copyRequest := *r
		copyURL := *r.URL
		copyURL.Path = "/" + asset
		copyURL.RawPath = ""
		copyRequest.URL = &copyURL
		a.static.ServeHTTP(w, &copyRequest)
		return
	}
	aliases := map[string]string{"/home": "/", "/about-us": "/meettheteam", "/about": "/meettheteam", "/about/": "/meettheteam", "/what-we-do": "/services", "/resources": "/patient-resources", "/client-specials": "/patient-resources", "/photo-gallery": "/visit-us", "/client-portal": "/patient-resources#client-portal", "/online-ordering": "/patient-resources#online-ordering", "/holiday-hours": "/visit-us#holiday-hours"}
	if target, ok := aliases[r.URL.Path]; ok {
		http.Redirect(w, r, target, http.StatusMovedPermanently)
		return
	}
	clean := path.Clean(r.URL.Path)
	if clean != r.URL.Path {
		http.Redirect(w, r, clean, http.StatusMovedPermanently)
		return
	}
	w.Header().Set("Cache-Control", "no-cache")
	p, ok := a.pages[r.URL.Path]
	if !ok {
		a.notFound(w, r)
		return
	}
	var buf bytes.Buffer
	if err := a.templates[r.URL.Path].ExecuteTemplate(&buf, "layout", p); err != nil {
		log.Printf("render %s: %v", p.Path, err)
		http.Error(w, "Could not display this page.", 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		_, _ = buf.WriteTo(w)
	}
}
func (a *application) notFound(w http.ResponseWriter, r *http.Request) {
	p := a.pages["/"]
	p.Title = "Page not found"
	var b bytes.Buffer
	if a.notFoundTemplate.ExecuteTemplate(&b, "layout", p) != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(404)
	if r.Method != http.MethodHead {
		_, _ = b.WriteTo(w)
	}
}
func main() {
	app, err := newApplication(os.Getenv("FINGERPRINT_ASSETS") == "1")
	if err != nil {
		log.Fatal(err)
	}
	port := 4173
	if value := os.Getenv("PORT"); value != "" {
		port, err = strconv.Atoi(value)
		if err != nil || port < 1 || port > 65535 {
			log.Fatal("PORT must be between 1 and 65535")
		}
	}
	host := os.Getenv("HOST")
	if host == "" {
		host = "127.0.0.1"
	}
	var listener net.Listener
	for attempt := 0; attempt < 10 && port <= 65535; attempt++ {
		listener, err = net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
		if err == nil {
			break
		}
		if !strings.Contains(err.Error(), "address already in use") {
			log.Fatal(err)
		}
		log.Printf("Port %d is in use; trying %d", port, port+1)
		port++
	}
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Eagle Vet preview: http://%s", listener.Addr())
	server := &http.Server{Handler: app, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	log.Fatal(server.Serve(listener))
}
