package fastboot

import (
	"bytes"
	"net/http"
	"path"
	"strings"

	"github.com/256dpi/serve"

	"github.com/256dpi/ember"
)

// Handler is a http.Handler that will pre-render the given ember app.
type Handler struct {
	app      *ember.App
	instance *Instance
}

// Handle will create a new handler.
func Handle(app *ember.App) (http.Handler, error) {
	// create instance
	instance, err := Boot(app)
	if err != nil {
		return nil, err
	}

	return &Handler{
		app:      app,
		instance: instance,
	}, nil
}

// ServeHTTP implements the http.Handler interface.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// check method
	if r.Method != "GET" {
		http.Error(w, "", http.StatusMethodNotAllowed)
		return
	}

	// remove leading and trailing slash
	pth := strings.Trim(r.URL.Path, "/")

	// handle exact matches
	file := h.app.File(pth)
	if file != nil {
		// set content type
		mimeType := serve.MimeTypeByExtension(path.Ext(pth), true)
		w.Header().Set("Content-Type", mimeType)

		// write file
		_, _ = w.Write(file)

		return
	}

	/* render requests */

	// clear URL prefix
	r.URL.Scheme = ""
	r.URL.Opaque = ""
	r.URL.Host = ""
	r.URL.User = nil

	// prepare index
	index := h.app.File("index.html")

	// set content type
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// visit URL
	result, err := h.instance.Visit(r.URL.String())
	if err != nil {
		// TODO: Log error.
		_, _ = w.Write(index)
		return
	}

	// apply attributes
	index = bytes.Replace(index, []byte("<body>"), []byte("<body"+result.BodyAttributesString()+">"), 1)
	index = bytes.Replace(index, []byte("<head>"), []byte("<head"+result.HeadAttributesString()+">"), 1)
	index = bytes.Replace(index, []byte("<html>"), []byte("<html"+result.HTMLAttributesString()+">"), 1)

	// wrap body with boundary tags
	body := `<script type="x/boundary" id="fastboot-body-start"></script>` + result.BodyContent + `<script type="x/boundary" id="fastboot-body-end"></script>`

	// replace content
	index = bytes.Replace(index, []byte("<!-- EMBER_CLI_FASTBOOT_HEAD -->"), []byte(result.HeadContent), 1)
	index = bytes.Replace(index, []byte("<!-- EMBER_CLI_FASTBOOT_BODY -->"), []byte(body), 1)

	// write result
	_, _ = w.Write(index)
}
