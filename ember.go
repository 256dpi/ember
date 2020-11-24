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

var indexHTMLFile = "index.html"
var headClosingTag = []byte("</head>")
var bodyClosingTag = []byte("</body>")

// App is an in-memory representation of an Ember.js application.
type App struct {
	files  map[string][]byte
	config map[string]interface{}
	before []byte
	env    []byte
	after  []byte
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
	index, ok := bytesFiles[indexHTMLFile]
	if !ok {
		return nil, fmt.Errorf("missing index file")
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

// Set will set the provided settings on the application.
func (a *App) Set(name string, value interface{}) {
	// set config
	a.config[name] = value

	// marshal config
	data, err := json.Marshal(a.config)
	if err != nil {
		panic(err)
	}

	// escape config (Ember.js uses decodeURIComponent)
	a.env = []byte(url.PathEscape(string(data)))

	// recompile
	a.recompile()
}

// AddInlineStyle will append the provided CSS at the end of the head tag.
func (a *App) AddInlineStyle(css string) {
	// inject style
	style := []byte("<style>" + css + "</style>\n</head>")
	a.after = bytes.Replace(a.after, headClosingTag, style, 1)

	// recompile
	a.recompile()
}

// AddInlineScript will append the provides JS at the end of the body tag.
func (a *App) AddInlineScript(js string) {
	// inject script
	script := []byte("<script>" + js + "</script>\n</body>")
	a.after = bytes.Replace(a.after, bodyClosingTag, script, 1)

	// recompile
	a.recompile()
}

func (a *App) recompile() {
	// prepare buffer
	index := make([]byte, len(a.before)+len(a.env)+len(a.after))

	// copy bytes
	copy(index, a.before)
	copy(index[len(a.before):], a.env)
	copy(index[len(a.before)+len(a.env):], a.after)

	// update index
	a.files[indexHTMLFile] = index
}

// IsPage will return whether the provided path matches a page.
func (a *App) IsPage(path string) bool {
	path = strings.Trim(path, "/")
	return path == indexHTMLFile || a.files[path] == nil
}

// IsAsset will return whether the provided path matches an asset.
func (a *App) IsAsset(path string) bool {
	path = strings.Trim(path, "/")
	return path != indexHTMLFile && a.files[path] != nil
}

// ServeHTTP implements the http.Handler interface.
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// check method
	if r.Method != "GET" {
		http.Error(w, "", http.StatusMethodNotAllowed)
		return
	}

	// remove leading and trailing slash
	pth := strings.Trim(r.URL.Path, "/")

	// get content
	content, ok := a.files[pth]
	if !ok {
		pth = indexHTMLFile
		content, _ = a.files[pth]
	}

	// get mime type
	mimeType := mime.TypeByExtension(path.Ext(pth))

	// set content type
	w.Header().Set("Content-Type", mimeType)

	// write file
	_, _ = w.Write(content)
}

// Handler will construct and return a dynamic handler that invokes the provided
// callback for each page request to allow dynamic configuration. If no dynamic
// configuration is needed, the app should be serve directly.
func (a *App) Handler(configure func(*App, *http.Request)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// handle assets
		if a.IsAsset(r.URL.Path) {
			a.ServeHTTP(w, r)
			return
		}

		// clone
		clone := a.Clone()

		// configure
		configure(clone, r)

		// serve
		clone.ServeHTTP(w, r)
	})
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
		env:    a.env,
		after:  a.after,
		config: config,
	}
}
