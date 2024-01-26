package fastboot

import (
	"context"
	_ "embed" // for embedding
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/log"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"

	"github.com/256dpi/ember"
)

//go:embed headers.js
var headersClass string

type manifest struct {
	Fastboot struct {
		AppName  string          `json:"appName"`
		Config   json.RawMessage `json:"config"`
		Manifest struct {
			AppFiles    []string `json:"appFiles"`
			HTMLFile    string   `json:"htmlFile"`
			VendorFiles []string `json:"vendorFiles"`
		} `json:"manifest"`
	} `json:"fastboot"`
}

// Request represents a request to be made.
type Request struct {
	Method      string              `json:"method"`
	Protocol    string              `json:"protocol"`
	Path        string              `json:"path"`
	Headers     map[string][]string `json:"headers"`
	Cookies     map[string]string   `json:"cookies"`
	QueryParams map[string]string   `json:"queryParams"`
	Body        string              `json:"body"`
}

// Result represents the result of an instance visit.
type Result struct {
	HeadContent    string            `json:"headContent"`
	BodyContent    string            `json:"bodyContent"`
	HTMLAttributes map[string]string `json:"htmlAttributes"`
	HeadAttributes map[string]string `json:"headAttributes"`
	BodyAttributes map[string]string `json:"bodyAttributes"`
}

// HTMLAttributesString will return the HTML attributes as a string.
func (r *Result) HTMLAttributesString() string {
	return attributesString(r.HTMLAttributes)
}

// HeadAttributesString will return the head attributes as a string.
func (r *Result) HeadAttributesString() string {
	return attributesString(r.HeadAttributes)
}

// BodyAttributesString will return the body attributes as a string.
func (r *Result) BodyAttributesString() string {
	return attributesString(r.BodyAttributes)
}

// HTML will build and return the full document.
func (r *Result) HTML() string {
	return fmt.Sprintf(`<!DOCTYPE html>
		<html%s>
			<head%s>
				%s
			</head>
			<body%s>
				%s
			</body>
		</html>
	`,
		r.HTMLAttributesString(),
		r.HeadAttributesString(),
		r.HeadContent,
		r.BodyAttributesString(),
		r.BodyContent,
	)
}

// TODO: Should we disable image downloads and CSS stuff?
//  => chromedp.Flag("blink-settings", "imagesEnabled=false")

// TODO: Use github.com/tidwall/go-node as a faster alternative to chromedp?

// TODO: Cache browser instance?
// TODO: Cache responses?
// TODO: Support for different manifests schemas?

// Instance represents a running Fastboot instance.
type Instance struct {
	app    *ember.App
	man    manifest
	ctx    context.Context
	cancel func()
	errs   []error
	mutex  sync.Mutex
}

// Boot will boot the provided Fastboot-capable app in a headless browser and
// return a running instance.
func Boot(app *ember.App, baseURL string) (*Instance, error) {
	// trim trailing slashes
	baseURL = strings.TrimRight(baseURL, "/")

	// get package.json file
	packageJSON := app.File("package.json")
	if packageJSON == nil {
		return nil, errors.New("missing package.json")
	}

	// parse manifest
	var manifest manifest
	err := json.Unmarshal(packageJSON, &manifest)
	if err != nil {
		return nil, fmt.Errorf("failed to parse package.json: %w", err)
	}

	// create context
	ctx, cancel := chromedp.NewContext(context.Background())

	// prepare instance
	instance := &Instance{
		app:    app,
		man:    manifest,
		ctx:    ctx,
		cancel: cancel,
	}

	// collect errors
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		if ev, ok := ev.(*fetch.EventRequestPaused); ok {
			go func() {
				if ev.Request.URL == baseURL+"/" {
					err := chromedp.Run(ctx,
						fetch.FulfillRequest(ev.RequestID, 200).
							WithBody(base64.StdEncoding.EncodeToString([]byte("<html><head></head><body></body></html>"))),
					)
					if err != nil {
						instance.errs = append(instance.errs, fmt.Errorf("%s (%s)", err.Error(), ev.Request.URL))
					}
				} else {
					err := chromedp.Run(ctx,
						fetch.ContinueRequest(ev.RequestID),
					)
					if err != nil {
						instance.errs = append(instance.errs, fmt.Errorf("%s (%s)", err.Error(), ev.Request.URL))
					}
				}
			}()
		}
		if ev, ok := ev.(*log.EventEntryAdded); ok {
			if ev.Entry.Level == log.LevelError {
				instance.errs = append(instance.errs, fmt.Errorf("%s (%s)", ev.Entry.Text, ev.Entry.URL))
			}
		}
	})

	// prepare actions
	var actions []chromedp.Action

	// enable fetch interception
	actions = append(actions, fetch.Enable())

	// open blank page
	actions = append(actions, chromedp.Navigate(baseURL))

	// prepare environment
	actions = append(actions, chromedp.Evaluate(`
		const config = `+string(instance.man.Fastboot.Config)+`;

		window.FastBoot = {
			config(name) {
				return config[name];
			},
			require(name) {
				if (name === 'crypto')  {
					return window.crypto;
				}
				if (name === 'node-fetch') {
					return {
						'default': window.fetch,
						FormData: window.FormData,
						Headers: window.Headers,
						Request: window.Request,
						Response: window.Response,
						FetchError: window.FetchError,
						AbortError: window.AbortError,
						isRedirect: window.isRedirect,
						Blob: window.Blob,
						File: window.File,
						fileFromSync: window.fileFromSync,
						fileFrom: window.fileFrom,
						blobFromSync: window.blobFromSync,
						blobFrom: window.blobFrom,
					};
				}
				if (name === 'abortcontroller-polyfill/dist/cjs-ponyfill') {
					return {
						AbortController: window.AbortController,
						AbortSignal: window.AbortSignal,
						fetch: window.fetch,
					};
				}
				return window.require(...arguments);
			},
		};
	`, nil))

	// evaluate scripts
	for _, file := range instance.man.Fastboot.Manifest.VendorFiles {
		actions = append(actions, chromedp.Evaluate(string(instance.app.File(file)), nil))
	}
	for _, file := range instance.man.Fastboot.Manifest.AppFiles {
		actions = append(actions, chromedp.Evaluate(string(instance.app.File(file)), nil))
	}

	// run application
	actions = append(actions, chromedp.Evaluate(`
		(async () => {
			window.$app = require('~fastboot/app-factory').default()
			await $app.boot();
		})()
	`, nil, func(params *runtime.EvaluateParams) *runtime.EvaluateParams {
		params.AwaitPromise = true
		return params
	}))

	// boot application
	err = chromedp.Run(instance.ctx, actions...)
	if err != nil {
		return nil, fmt.Errorf("failed to boot application: %w", err)
	}

	return instance, nil
}

// Visit will visit the provided URL and return the result.
func (i *Instance) Visit(relURL string, r Request) (Result, error) {
	// acquire mutex
	i.mutex.Lock()
	defer i.mutex.Unlock()

	// prepare actions
	var actions []chromedp.Action

	// marshal request
	data, err := json.Marshal(r)
	if err != nil {
		return Result{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	// run application
	actions = append(actions, chromedp.Evaluate(`
		(async () => {
			`+headersClass+`
	
			if (window.$instance) {
				await $instance.destroy();
			}
	
			let removeAttributes = (node) => {
				while(node.attributes.length > 0) {
					node.removeAttribute(node.attributes[0].name);
				}
			}
	
			document.head.innerHTML = '';
			document.body.innerHTML = '';
			removeAttributes(document.head);
			removeAttributes(document.body);
			removeAttributes(document.documentElement);
	
			window.$instance = await $app.buildInstance();
	
			const request = `+string(data)+`;
			request.headers = new FastBootHeaders(request.headers);
			request.host = () => request.headers.get('host');
	
			const info = {
				request: request,
				response: {
					headers: new FastBootHeaders({}),
					statusCode: 200,
				},
				metadata: {},
				deferredPromise: Promise.resolve(),
				deferRendering(promise) {
					this.deferredPromise = promise;
				},
			};
	
			$instance.register('info:-fastboot', info, { instantiate: false });
	
			const options = {
				document: window.document,
				isBrowser: true,
				isInteractive: false,
				rootElement: window.document.body,
			};
	
			await $instance.boot(options);
			await $instance.visit('`+relURL+`', options);
			await info.deferredPromise;
	
			return info;
		})()
	`, nil, func(params *runtime.EvaluateParams) *runtime.EvaluateParams {
		params.AwaitPromise = true
		return params
	}))

	// capture HTML
	var result Result
	actions = append(actions, chromedp.Evaluate(`({
		headContent: document.head.innerHTML,
		bodyContent: document.body.innerHTML,
		htmlAttributes: Object.fromEntries(Array.from(document.documentElement.attributes).map(a => [a.name, a.value])),
		headAttributes: Object.fromEntries(Array.from(document.head.attributes).map(a => [a.name, a.value])),
		bodyAttributes: Object.fromEntries(Array.from(document.body.attributes).map(a => [a.name, a.value])),
	})`, &result))

	// run application
	err = chromedp.Run(i.ctx, actions...)
	if err != nil {
		return Result{}, fmt.Errorf("failed to visit URL: %w", err)
	}

	// handle errors
	if len(i.errs) > 0 {
		err = errors.Join(i.errs...)
		i.errs = nil
		return Result{}, fmt.Errorf("failed to visit URL: %w", err)
	}

	return result, nil
}

// Close will close the instance and release all resources.
func (i *Instance) Close() {
	// acquire mutex
	i.mutex.Lock()
	defer i.mutex.Unlock()

	// cancel context
	i.cancel()

	// clear context
	i.ctx = nil
	i.cancel = nil
}

// Render will run the provided app in a headless browser and return the HTML
// output for the specified URL.
func Render(app *ember.App, absURL string, r Request) (Result, error) {
	// parse URL
	_url, err := url.Parse(absURL)
	if err != nil {
		return Result{}, err
	}

	// determine base URL
	baseURL := fmt.Sprintf("%s://%s", _url.Scheme, _url.Host)

	// determine relative URL
	_url.Scheme = ""
	_url.Opaque = ""
	_url.Host = ""
	relURL := _url.String()

	// boot app
	instance, err := Boot(app, baseURL)
	if err != nil {
		return Result{}, err
	}
	defer instance.Close()

	// visit URL
	result, err := instance.Visit(relURL, r)
	if err != nil {
		return Result{}, err
	}

	return result, nil
}

func attributesString(attrs map[string]string) string {
	var result string
	for name, value := range attrs {
		result += fmt.Sprintf(` %s="%s"`, name, value)
	}
	return result
}
