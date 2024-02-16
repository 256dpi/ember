package fastboot

import (
	"bytes"
	"net/http"
	"strings"
	"time"

	gocache "github.com/patrickmn/go-cache"

	"github.com/256dpi/ember"
)

// Options are used to configure the handler.
type Options struct {
	App       *ember.App
	Origin    string
	Timeout   time.Duration // 5s
	Cache     time.Duration
	Isolated  bool
	Headed    bool
	OnRequest func(*Request)
	OnResult  func(*Result)
	OnError   func(error)
}

// Handler is a http.Handler that will pre-render the given ember app.
type Handler struct {
	options  Options
	cache    *gocache.Cache
	instance *Instance
}

// Handle will create a new handler.
func Handle(options Options) (*Handler, error) {
	// ensure timeout
	if options.Timeout == 0 {
		options.Timeout = 5 * time.Second
	}

	// prepare cache
	var cache *gocache.Cache
	if options.Cache > 0 {
		cache = gocache.New(options.Cache, options.Cache/4)
	}

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
		cache:    cache,
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

	// handle static files
	if h.options.App.File(pth) != nil {
		h.options.App.ServeHTTP(w, r)
		return
	}

	/* render requests */

	// set content type
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// serve cached result if possible
	if h.cache != nil {
		cached, ok := h.cache.Get(pth)
		if ok {
			http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(cached.([]byte)))
			return
		}
	}

	// build request
	request := Request{
		Method:   "GET",
		Protocol: r.URL.Scheme,
		Path:     r.URL.Path,
		Headers: map[string][]string{
			"Host": {r.Host},
		},
		Cookies:     map[string]string{},
		QueryParams: map[string]string{},
		Body:        "",
	}

	// clear URL prefix
	r.URL.Scheme = ""
	r.URL.Opaque = ""
	r.URL.Host = ""
	r.URL.User = nil

	// prepare index
	index := h.options.App.File("index.html")

	// prepare instance
	instance := h.instance
	if instance == nil {
		var err error
		instance, err = Boot(h.options.App, h.options.Origin, h.options.Headed)
		if err != nil {
			if h.options.OnError != nil {
				h.options.OnError(err)
			}
			http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(index))
			return
		}
		defer instance.Close()
	}

	// call request handler
	if h.options.OnRequest != nil {
		h.options.OnRequest(&request)
	}

	// visit URL
	result, err := instance.Visit(r.URL.String(), request, h.options.Timeout)
	if err != nil {
		if h.options.OnError != nil {
			h.options.OnError(err)
		}
		http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(index))
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
	index = bytes.Replace(index, []byte("<!-- EMBER_CLI_FASTBOOT_TITLE -->"), nil, 1)
	index = bytes.Replace(index, []byte("<!-- EMBER_CLI_FASTBOOT_HEAD -->"), []byte(result.HeadContent), 1)
	index = bytes.Replace(index, []byte("<!-- EMBER_CLI_FASTBOOT_BODY -->"), []byte(body), 1)

	// write result
	http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(index))

	// cache result if possible
	if h.cache != nil {
		h.cache.Set(pth, index, h.options.Cache)
	}
}

// Close will close the handler.
func (h *Handler) Close() {
	if h.instance != nil {
		h.instance.Close()
	}
}
