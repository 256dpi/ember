package fastboot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/chromedp/cdproto/log"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"

	"github.com/256dpi/ember"
)

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

// Result represents the result of a Fastboot visit.
type Result struct {
	HeadContent    string            `json:"headContent"`
	BodyContent    string            `json:"bodyContent"`
	HTMLAttributes map[string]string `json:"htmlAttributes"`
	HeadAttributes map[string]string `json:"headAttributes"`
	BodyAttributes map[string]string `json:"bodyAttributes"`
}

// HTML will return the full HTML.
func (r *Result) HTML() string {
	attributes := func(attrs map[string]string) string {
		var result string
		for name, value := range attrs {
			result += fmt.Sprintf(` %s="%s"`, name, value)
		}
		return result
	}

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
		attributes(r.HTMLAttributes),
		attributes(r.HeadAttributes),
		r.HeadContent,
		attributes(r.BodyAttributes),
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
}

// Boot will boot the provided app in a headless browser and return a running
// instance.
func Boot(app *ember.App) (*Instance, error) {
	// parse manifest
	var manifest manifest
	err := json.Unmarshal(app.File("package.json"), &manifest)
	if err != nil {
		return nil, err
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
		if ev, ok := ev.(*log.EventEntryAdded); ok {
			if ev.Entry.Level == log.LevelError {
				instance.errs = append(instance.errs, fmt.Errorf("%s (%s)", ev.Entry.Text, ev.Entry.URL))
			}
		}
	})

	// prepare actions
	var actions []chromedp.Action

	// open blank page
	actions = append(actions, chromedp.Navigate("about:blank"))

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

	// prepare application
	err = chromedp.Run(instance.ctx, actions...)
	if err != nil {
		return nil, err
	}

	return instance, nil
}

// Visit will run the provided app in a headless browser and return the rendered
// HTML for the specified URL.
func (i *Instance) Visit(url string) (Result, error) {
	// prepare actions
	var actions []chromedp.Action

	// run application
	actions = append(actions, chromedp.Evaluate(`
		(async () => {
			if (window.$instance) {
				await $instance.destroy();
			}
	
			window.$instance = await $app.buildInstance();
	
			const info = {
				request: {
					method: 'GET',
					path: '`+url+`',
					headers: {},
					cookies: {},
					queryParams: {},
					body: null,
				},
				response: {
					headers: {},
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
				isBrowser: false,
				rootElement: window.document.body,
			};
	
			await $instance.boot(options);
			await $instance.visit('`+url+`', options);
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

	// render application
	err := chromedp.Run(i.ctx, actions...)
	if err != nil {
		return Result{}, err
	}

	// handle errors
	if len(i.errs) > 0 {
		err = errors.Join(i.errs...)
		i.errs = nil
		return Result{}, err
	}

	return result, nil
}

// Close will close the instance and release all resources.
func (i *Instance) Close() {
	// cancel context
	i.cancel()

	// clear context
	i.ctx = nil
	i.cancel = nil
}

// Render will run the provided app in a headless browser and return the HTML
// output for the specified URL.
func Render(app *ember.App, url string) (Result, error) {
	// boot app
	instance, err := Boot(app)
	if err != nil {
		return Result{}, err
	}
	defer instance.Close()

	// visit URL
	result, err := instance.Visit(url)
	if err != nil {
		return Result{}, err
	}

	return result, nil
}
