package fastboot

import (
	"bytes"
	"net/http"
	"path"
	"strings"

	"github.com/256dpi/serve"

	"github.com/256dpi/ember"
)

// Options are used to configure the handler.
type Options struct {
	App       *ember.App
	Origin    string
	Isolated  bool
	Headed    bool
	OnRequest func(*Request)
	OnResult  func(*Result)
	OnError   func(error)
}

// Handler is a http.Handler that will pre-render the given ember app.
type Handler struct {
	options  Options
	instance *Instance
}

// Handle will create a new handler.
func Handle(options Options) (*Handler, error) {
	// create instance
	var instance *Instance
	if !options.Isolated {
		var err error
		instance, err = Boot(options.App, options.Origin, options.Headed)
		if err != nil {
			return nil, err
		}
	}

	return &Handler{
		options:  options,
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
	file := h.options.App.File(pth)
	if file != nil {
		// set content type
		mimeType := serve.MimeTypeByExtension(path.Ext(pth), true)
		w.Header().Set("Content-Type", mimeType)

		// write file
		_, _ = w.Write(file)

		return
	}

	/* render requests */

	// build request
	request := Request{
		Method:      "GET",
		Protocol:    r.URL.Scheme,
		Path:        r.URL.Path,
		Headers:     r.Header,
		Cookies:     map[string]string{}, // TODO: Set.
		QueryParams: map[string]string{}, // TODO: Set.
		Body:        "",
	}

	// clear URL prefix
	r.URL.Scheme = ""
	r.URL.Opaque = ""
	r.URL.Host = ""
	r.URL.User = nil

	// prepare index
	index := h.options.App.File("index.html")

	// set content type
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// prepare instance
	instance := h.instance
	if instance == nil {
		var err error
		instance, err = Boot(h.options.App, h.options.Origin, h.options.Headed)
		if err != nil {
			if h.options.OnError != nil {
				h.options.OnError(err)
			}
			_, _ = w.Write(index)
			return
		}
		defer instance.Close()
	}

	// call request handler
	if h.options.OnRequest != nil {
		h.options.OnRequest(&request)
	}

	// visit URL
	result, err := instance.Visit(r.URL.String(), request)
	if err != nil {
		if h.options.OnError != nil {
			h.options.OnError(err)
		}
		_, _ = w.Write(index)
		return
	}

	// call result handler
	if h.options.OnResult != nil {
		h.options.OnResult(&result)
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

// Close will close the handler.
func (h *Handler) Close() {
	if h.instance != nil {
		h.instance.Close()
	}
}
