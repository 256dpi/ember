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
	parent *App
	files  map[string][]byte
	config map[string]interface{}
	index  [3][]byte
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

	// get chunks
	head := index[:start+len(tagStart)]
	meta := index[start+len(tagStart) : start+len(tagStart)+end]
	tail := index[start+len(tagStart)+end:]

	// unescape meta
	data, err := url.QueryUnescape(string(meta))
	if err != nil {
		return nil, err
	}

	// unmarshal configuration
	config := map[string]interface{}{}
	err = json.Unmarshal([]byte(data), &config)
	if err != nil {
		return nil, err
	}

	return &App{
		files:  bytesFiles,
		index:  [3][]byte{head, meta, tail},
		config: config,
	}, nil
}

// Set will set the provided settings on the application.
func (a *App) Set(name string, value interface{}) {
	// copy config if missing
	a.copyConfig()

	// set config
	a.config[name] = value

	// marshal config
	data, err := json.Marshal(a.config)
	if err != nil {
		panic(err)
	}

	// escape config (Ember.js uses decodeURIComponent)
	a.index[1] = []byte(url.PathEscape(string(data)))

	// recompile
	a.recompile()
}

// AddInlineStyle will append the provided CSS at the end of the head tag.
func (a *App) AddInlineStyle(css string) {
	// inject style
	style := []byte("<style>" + css + "</style>\n</head>")
	a.index[2] = bytes.Replace(a.index[2], headClosingTag, style, 1)

	// recompile
	a.recompile()
}

// AddInlineScript will append the provides JS at the end of the body tag.
func (a *App) AddInlineScript(js string) {
	// inject script
	script := []byte("<script>" + js + "</script>\n</body>")
	a.index[2] = bytes.Replace(a.index[2], bodyClosingTag, script, 1)

	// recompile
	a.recompile()
}

func (a *App) recompile() {
	// copy files if missing
	a.copyFiles()

	// prepare buffer
	buffer := make([]byte, len(a.index[0])+len(a.index[1])+len(a.index[2]))

	// copy bytes
	copy(buffer, a.index[0])
	copy(buffer[len(a.index[0]):], a.index[1])
	copy(buffer[len(a.index[0])+len(a.index[1]):], a.index[2])

	// update index
	a.files[indexHTMLFile] = buffer
}

// IsPage will return whether the provided path matches a page.
func (a *App) IsPage(path string) bool {
	path = strings.Trim(path, "/")
	return path == indexHTMLFile || a.getFiles()[path] == nil
}

// IsAsset will return whether the provided path matches an asset.
func (a *App) IsAsset(path string) bool {
	path = strings.Trim(path, "/")
	return path != indexHTMLFile && a.getFiles()[path] != nil
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
	content, ok := a.getFiles()[pth]
	if !ok {
		pth = indexHTMLFile
		content, _ = a.getFiles()[pth]
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
	return &App{
		parent: a,
		index:  a.index,
	}
}

func (a *App) getFiles() map[string][]byte {
	// check files
	if a.files != nil {
		return a.files
	}

	return a.parent.getFiles()
}

func (a *App) copyFiles() {
	if a.files == nil {
		parent := a.getFiles()
		a.files = map[string][]byte{}
		for key, value := range parent {
			a.files[key] = value
		}
	}
}

func (a *App) getConfig() map[string]interface{} {
	// check config
	if a.config != nil {
		return a.config
	}

	return a.parent.getConfig()
}

func (a *App) copyConfig() {
	if a.config == nil {
		parent := a.getConfig()
		a.config = map[string]interface{}{}
		for key, value := range parent {
			a.config[key] = value
		}
	}
}
