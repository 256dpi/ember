// Package ember provides tools to serve Ember.js apps from Go HTTP handlers.
package ember

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strings"
)

// Config represents an Ember.js app configuration.
type Config = map[string]interface{}

// App is an in-memory representation of an Ember.js application.
type App struct {
	name   string
	files  map[string][]byte
	before []byte
	after  []byte
	config Config
}

// MustCreate will call Create and panic on errors.
func MustCreate(name string, build map[string]string) *App {
	// create app
	app, err := Create(name, build)
	if err != nil {
		panic(err)
	}

	return app
}

// Create will create and Ember.js application instance from the provided build.
func Create(name string, build map[string]string) (*App, error) {
	// convert files
	files := make(map[string][]byte)
	for file, content := range build {
		files[file] = []byte(content)
	}

	// get index
	index, ok := files["index.html"]
	if !ok {
		return nil, fmt.Errorf("missing index.html")
	}

	// compute tag start and end
	tagStart := fmt.Sprintf(`<meta name="%s/config/environment" content="`, name)
	tagEnd := `"/>`

	// find tag start
	start := bytes.Index(index, []byte(tagStart))
	if start < 0 {
		return nil, fmt.Errorf("config meta tag start not found")
	}

	// find tag end
	end := bytes.Index(index[start+len(tagStart):], []byte(tagEnd))
	if end < 0 {
		return nil, fmt.Errorf("config meta tag end not found")
	}

	// get meta
	meta := index[start+len(tagStart) : start+len(tagStart)+end]

	// unescape attribute
	data, err := url.QueryUnescape(string(meta))
	if err != nil {
		return nil, err
	}

	// unmarshal configuration
	var config Config
	err = json.Unmarshal([]byte(data), &config)
	if err != nil {
		return nil, err
	}

	return &App{
		name:   name,
		files:  files,
		before: index[:start+len(tagStart)],
		after:  index[start+len(tagStart)+end:],
		config: config,
	}, nil
}

// MustSet will call Set and panic on errors.
func (a *App) MustSet(name string, value interface{}) {
	// set value
	err := a.Set(name, value)
	if err != nil {
		panic(err)
	}
}

// Set will set the provided settings on the application.
func (a *App) Set(name string, value interface{}) error {
	// set config
	a.config[name] = value

	// marshal config
	data, err := json.Marshal(a.config)
	if err != nil {
		return err
	}

	// escape config
	data = []byte(url.QueryEscape(string(data)))

	// prepare index
	index := make([]byte, len(a.before)+len(data)+len(a.after))

	// copy bytes
	copy(index, a.before)
	copy(index[len(a.before):], data)
	copy(index[len(a.before)+len(data):], a.after)

	// update index
	a.files["index.html"] = index

	return nil
}

// ServeHTTP implements the http.Handler interface.
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// check method
	if r.Method != "GET" {
		http.Error(w, "", http.StatusMethodNotAllowed)
		return
	}

	// get path
	pth := strings.TrimPrefix(r.URL.Path, "/")

	// get content
	content, ok := a.files[pth]
	if !ok {
		pth = "index.html"
		content, _ = a.files[pth]
	}

	// get mime type
	mimeType := mime.TypeByExtension(path.Ext(pth))

	// set content type
	w.Header().Set("Content-Type", mimeType)

	// write file
	_, _ = w.Write(content)
}

// Clone will make a copy of the application.
func (a *App) Clone() *App {
	// clone files
	files := map[string][]byte{}
	for file, content := range a.files {
		files[file] = content
	}

	// clone config
	config := Config{}
	for key, value := range a.config {
		config[key] = value
	}

	return &App{
		name:   a.name,
		files:  files,
		before: a.before,
		after:  a.after,
		config: config,
	}
}
