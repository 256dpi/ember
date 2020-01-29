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

// App is an in-memory representation of an Ember.js application.
type App struct {
	files  map[string][]byte
	before []byte
	after  []byte
	config map[string]interface{}
}

// MustCreate will call Create and panic on errors.
func MustCreate(name string, files map[string]string) *App {
	// create app
	app, err := Create(name, files)
	if err != nil {
		panic(err)
	}

	return app
}

// Create will create and Ember.js application instance from the provided files.
// The provided map must at least include the "index.html" key with the contents
// of the index html file. All other files e.g. "assets/app.css" are served with
// their corresponding MIME types read from the file extension.
func Create(name string, files map[string]string) (*App, error) {
	// convert files
	bytesFiles := make(map[string][]byte)
	for file, content := range files {
		bytesFiles[file] = []byte(content)
	}

	// get index
	index, ok := bytesFiles["index.html"]
	if !ok {
		return nil, fmt.Errorf("missing index.html")
	}

	// find tag start
	tagStart := fmt.Sprintf(`<meta name="%s/config/environment" content="`, name)
	start := bytes.Index(index, []byte(tagStart))
	if start < 0 {
		return nil, fmt.Errorf("config meta tag start not found")
	}

	// find attribute end
	end := bytes.Index(index[start+len(tagStart):], []byte(`"`))
	if end < 0 {
		return nil, fmt.Errorf("config meta tag end not found")
	}

	// get meta content
	content := index[start+len(tagStart) : start+len(tagStart)+end]

	// unescape content
	data, err := url.QueryUnescape(string(content))
	if err != nil {
		return nil, err
	}

	// unmarshal configuration
	var config map[string]interface{}
	err = json.Unmarshal([]byte(data), &config)
	if err != nil {
		return nil, err
	}

	return &App{
		files:  bytesFiles,
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

	// remove leading slash
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
	config := map[string]interface{}{}
	for key, value := range a.config {
		config[key] = value
	}

	return &App{
		files:  files,
		before: a.before,
		after:  a.after,
		config: config,
	}
}
