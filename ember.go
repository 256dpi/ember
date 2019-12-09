package serve

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

// Config represents an Ember.js app configuration.
type Config = map[string]interface{}

// App is an in-memory representation of an Ember.js application.
type App struct {
	name   string
	files  map[string][]byte
	index  *goquery.Document
	meta   *goquery.Selection
	config Config
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

	// parse document
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(index))
	if err != nil {
		return nil, err
	}

	// find tag
	meta := doc.Find(fmt.Sprintf("meta[name='%s/config/environment']", name))
	if meta.Length() == 0 {
		return nil, fmt.Errorf("config meta tag not found")
	}

	// get attribute
	attr, ok := meta.Attr("content")
	if !ok {
		return nil, fmt.Errorf(`missing "content" attribute`)
	}

	// unescape attribute
	data, err := url.QueryUnescape(attr)
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
		index:  doc,
		meta:   meta,
		config: config,
	}, nil
}

// Set will set the provided settings on the application.
func (a *App) Set(name string, value string) error {
	// set config
	a.config[name] = value

	// marshal config
	data, err := json.Marshal(a.config)
	if err != nil {
		return err
	}

	// escape and set attribute
	a.meta.SetAttr("content", url.QueryEscape(string(data)))

	// render document
	var buf bytes.Buffer
	err = html.Render(&buf, a.index.Nodes[0])
	if err != nil {
		return err
	}

	// update index
	a.files["index.html"] = buf.Bytes()

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

	// clone index
	index := goquery.CloneDocument(a.index)

	// find tag
	meta := index.Find(fmt.Sprintf("meta[name='%s/config/environment']", a.name))
	if meta.Length() == 0 {
		panic(fmt.Errorf("config meta tag not found"))
	}

	return &App{
		name:   a.name,
		files:  files,
		index:  index,
		meta:   meta,
		config: config,
	}
}
