package fastboot

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/chromedp/cdproto/dom"
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
	
			window.$instance = await $app.buildInstance();
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
func (i *Instance) Visit(url string) (string, error) {
	// prepare actions
	var actions []chromedp.Action

	// run application
	actions = append(actions, chromedp.Evaluate(`
		(async () => {
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
	
			// $instance.destroy();
	
			return info;
		})()
	`, nil, func(params *runtime.EvaluateParams) *runtime.EvaluateParams {
		params.AwaitPromise = true
		return params
	}))

	// capture HTML
	var html string
	actions = append(actions, chromedp.ActionFunc(func(ctx context.Context) error {
		// get root
		node, err := dom.GetDocument().Do(ctx)
		if err != nil {
			return err
		}

		// capture HTML
		html, err = dom.GetOuterHTML().WithNodeID(node.NodeID).Do(ctx)
		if err != nil {
			return err
		}

		return nil
	}))

	// render application
	err := chromedp.Run(i.ctx, actions...)
	if err != nil {
		return "", err
	}

	// handle log errors
	if len(i.errs) > 0 {
		err = fmt.Errorf("log errors: %s", i.errs)
		i.errs = nil
		return "", err
	}

	return html, nil
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
func Render(app *ember.App, url string) (string, error) {
	// boot app
	instance, err := Boot(app)
	if err != nil {
		return "", err
	}
	defer instance.Close()

	// visit URL
	html, err := instance.Visit(url)
	if err != nil {
		return "", err
	}

	return html, nil
}
