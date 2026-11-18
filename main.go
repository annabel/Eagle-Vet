package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
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
	ID, Title, Intro, Image, ImageAlt string
	Items                             []string
	More                              []Detail
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
	pages     map[string]Page
	templates map[string]*template.Template
	static    http.Handler
}

func loadJSON(name string, target any) error {
	b, err := files.ReadFile("content/" + name + ".json")
	if err != nil {
		return err
	}
	return json.Unmarshal(b, target)
}

func newApplication() (*application, error) {
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
		"/services":          {Name: "Services", Title: "Our Services", Heading: "A little care goes", Accent: "a long way.", Intro: "From everyday wellness to the unexpected, personalized veterinary care for the Treasure Valley’s dogs, cats, and exotic companions.", Image: "/assets/services-hero.jpg", ImageAlt: "A cream-colored puppy sitting in the grass", Template: "services"},
		"/testimonials":      {Name: "Testimonials", Title: "Kind Words from Our Clients", Heading: "Happy pets and the", Accent: "people who love them", Intro: "The relationships we build are at the heart of what we do. Here’s what a few of our pet families have shared about their visits.", Image: "/assets/testimonials-hero-boat.jpg", ImageAlt: "A black-and-white dog sitting on a boat at sunset", Template: "testimonials"},
		"/patient-resources": {Name: "Patient Resources", Title: "Patient Resources", Heading: "Everything you need,", Accent: "a little closer.", Intro: "Find your pet’s client portal and order food and medication online.", Image: "/assets/patient-resources-puppy.png", ImageAlt: "A cream-colored puppy lying down with one paw stretched out", Template: "resources"},
		"/visit-us":          {Name: "Visit Us", Title: "Visit & Contact Us", Heading: "Come visit us at our log cabin.", Intro: "Find us on Palmetto Avenue in Eagle. Whether you’re new here or already part of the family, we’d love to hear from you.", Image: "/assets/cabin-demo.png", ImageAlt: "Eagle Veterinary Hospital’s log cabin on Palmetto Avenue", Template: "visit"},
		"/appointment":       {Name: "Appointment", Title: "Request an Appointment", Template: "appointment"},
		"/hello":             {Name: "Hello", Title: "Welcome to Eagle Vet", Template: "hello"},
	}
	a := &application{pages: pages, templates: map[string]*template.Template{}}
	for route, p := range pages {
		p.Path = route
		p.Nav = nav
		p.Staff = staff
		p.Services = services
		p.Quotes = quotes
		p.Year = time.Now().Year()
		a.pages[route] = p
		t, err := template.ParseFS(files, "templates/layout.html", "templates/partials/*.html", "templates/pages/"+p.Template+".html")
		if err != nil {
			return nil, fmt.Errorf("template %s: %w", route, err)
		}
		a.templates[route] = t
	}
	staticFS, err := fs.Sub(files, "static")
	if err != nil {
		return nil, err
	}
	a.static = http.FileServer(http.FS(staticFS))
	return a, nil
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
		asset := strings.TrimPrefix(r.URL.Path, "/")
		if strings.HasPrefix(asset, "static/") {
			asset = strings.TrimPrefix(asset, "static/")
		}
		if strings.Contains(asset, "..") || strings.Contains(asset, "/.") {
			http.NotFound(w, r)
			return
		}
		if _, err := fs.Stat(files, "static/"+asset); err != nil {
			http.NotFound(w, r)
			return
		}
		copyRequest := r.Clone(r.Context())
		copyURL := *r.URL
		copyURL.Path = "/" + asset
		copyRequest.URL = &copyURL
		a.static.ServeHTTP(w, copyRequest)
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
	t, err := template.ParseFS(files, "templates/layout.html", "templates/partials/*.html", "templates/pages/404.html")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	var b bytes.Buffer
	if t.ExecuteTemplate(&b, "layout", p) != nil {
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
	app, err := newApplication()
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
