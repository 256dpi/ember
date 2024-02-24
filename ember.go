// Package ember provides tools to serve Ember.js apps from Go HTTP handlers.
package ember

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/256dpi/serve"
)

var indexHTMLFile = "index.html"
var headOpeningTag = []byte("<head>")
var headClosingTag = []byte("</head>")
var bodyClosingTag = []byte("</body>")

// App is an in-memory representation of an Ember.js application.
type App struct {
	name     string
	parent   *App
	files    map[string][]byte
	index    [3][]byte
	config   map[string]interface{}
	modified time.Time
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
		name:     name,
		files:    bytesFiles,
		index:    [3][]byte{head, meta, tail},
		config:   config,
		modified: time.Now(),
	}, nil
}

// Name will return the name of the application.
func (a *App) Name() string {
	return a.name
}

// Config will return the configuration of the application.
func (a *App) Config() map[string]interface{} {
	return a.getConfig()
}

// Get will get the specified setting from the application.
func (a *App) Get(name string) interface{} {
	return a.getConfig()[name]
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
	a.AppendHead("<style>" + css + "</style>")
}

// PrependHead will prepend the provided tag to the head tag.
func (a *App) PrependHead(tag string) {
	// inject tag
	a.index[0] = bytes.Replace(a.index[0], headOpeningTag, []byte("<head>\n"+tag), 1)

	// recompile
	a.recompile()
}

// AppendHead will append the provided tag to the head tag.
func (a *App) AppendHead(tag string) {
	// inject tag
	a.index[2] = bytes.Replace(a.index[2], headClosingTag, []byte(tag+"\n</head>"), 1)

	// recompile
	a.recompile()
}

// AddInlineScript will append the provides JS at the end of the body tag.
func (a *App) AddInlineScript(js string) {
	a.AppendBody("<script>" + js + "</script>")
}

// AppendBody will append the provided tag to the body tag.
func (a *App) AppendBody(tag string) {
	// inject tag
	a.index[2] = bytes.Replace(a.index[2], bodyClosingTag, []byte(tag+"\n</body>"), 1)

	// recompile
	a.recompile()
}

// Prefix will change the root URL and prefix all assets paths with the
// specified prefix. The app must be served with http.StripPrefix() to work
// correctly. If fixCSS is set to true, the app will also prefix all CSS
// url() paths. If dirs is empty or nil, the default "assets" directory will
// be used.
func (a *App) Prefix(prefix string, dirs []string, fixCSS bool) {
	// ensure default dirs
	if dirs == nil {
		dirs = []string{"assets"}
	}

	// cleanup prefix and dirs
	prefix = "/" + strings.Trim(prefix, "/")
	for i, dir := range dirs {
		dirs[i] = strings.Trim(dir, "/")
	}

	// set root url
	a.Set("rootURL", prefix+"/")

	// prefix index paths
	for _, dir := range dirs {
		a.index[0] = bytes.Replace(a.index[0], []byte(`src="/`+dir+`/`), []byte(`src="`+prefix+`/`+dir+`/`), -1)
		a.index[2] = bytes.Replace(a.index[2], []byte(`src="/`+dir+`/`), []byte(`src="`+prefix+`/`+dir+`/`), -1)
		a.index[2] = bytes.Replace(a.index[2], []byte(`href="/`+dir+`/`), []byte(`href="`+prefix+`/`+dir+`/`), -1)

	}

	// recompile
	a.recompile()

	// prefix other files
	for name, file := range a.files {
		// skip index
		if name == indexHTMLFile {
			continue
		}

		// prefix .html files
		if strings.HasSuffix(name, ".html") {
			for _, dir := range dirs {
				file = bytes.Replace(file, []byte(`src="/`+dir+`/`), []byte(`src="`+prefix+`/`+dir+`/`), -1)
				file = bytes.Replace(file, []byte(`href="/`+dir+`/`), []byte(`href="`+prefix+`/`+dir+`/`), -1)
			}
		}

		// prefix .css files
		if fixCSS && strings.HasSuffix(name, ".css") {
			for _, dir := range dirs {
				file = bytes.Replace(file, []byte(`url(/`+dir+`/`), []byte(`url(`+prefix+`/`+dir+`/`), -1)
				file = bytes.Replace(file, []byte(`url("/`+dir+`/`), []byte(`url("`+prefix+`/`+dir+`/`), -1)
			}
		}

		// replace file
		a.files[name] = file
	}
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

	// update modified
	a.modified = time.Now()
}

// AddFile will add the specified file to the app.
func (a *App) AddFile(name string, contents string) {
	// copy files if missing
	a.copyFiles()

	// set file
	a.files[name] = []byte(contents)

	// update modified
	a.modified = time.Now()
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

// File returns the contents of the specified file.
func (a *App) File(path string) []byte {
	return a.getFiles()[path]
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

	// set content type
	mimeType := serve.MimeTypeByExtension(path.Ext(pth), true)
	w.Header().Set("Content-Type", mimeType)

	// serve file
	http.ServeContent(w, r, pth, a.modified, bytes.NewReader(content))
}

// Handler will construct and return a dynamic handler that invokes the provided
// callback for each page request to allow dynamic configuration. If no dynamic
// configuration is needed, the app should be served directly.
func (a *App) Handler(configure func(*App, *http.Request)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// handle assets
		if a.IsAsset(r.URL.Path) {
			a.ServeHTTP(w, r)
			return
		}

		// configure clone
		clone := a.Clone()
		configure(clone, r)

		// serve
		clone.ServeHTTP(w, r)
	})
}

// Clone will make a copy of the application.
func (a *App) Clone() *App {
	return &App{
		name:   a.name,
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
