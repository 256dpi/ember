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

// Visit will run the provided app in a headless browser and return the rendered
// HTML for the specified URL.
func Visit(ctx context.Context, app *ember.App, url string) (string, error) {
	// parse manifest
	var manifest manifest
	err := json.Unmarshal(app.File("package.json"), &manifest)
	if err != nil {
		return "", err
	}

	// ensure context
	if ctx == nil {
		var cancel func()
		ctx, cancel = chromedp.NewContext(ctx)
		defer cancel()
	}

	// collect errors
	var logErrors []string
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		if ev, ok := ev.(*log.EventEntryAdded); ok {
			if ev.Entry.Level == log.LevelError {
				logErrors = append(logErrors, fmt.Sprintf("%s (%s)", ev.Entry.Text, ev.Entry.URL))
			}
		}
	})

	// prepare actions
	var actions []chromedp.Action

	// prepare environment
	actions = append(actions, chromedp.Evaluate(`
		const config = `+string(manifest.Fastboot.Config)+`;

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
	for _, file := range manifest.Fastboot.Manifest.VendorFiles {
		actions = append(actions, chromedp.Evaluate(string(app.File(file)), nil))
	}
	for _, file := range manifest.Fastboot.Manifest.AppFiles {
		actions = append(actions, chromedp.Evaluate(string(app.File(file)), nil))
	}

	// run application
	actions = append(actions, chromedp.Evaluate(`
		(async () => {
			const app = require('~fastboot/app-factory').default()
			await app.boot();
	
			const instance = await app.buildInstance();
	
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
	
			instance.register('info:-fastboot', info, { instantiate: false });
	
			const options = {
				document: window.document,
				isBrowser: false,
				rootElement: window.document.body,
			};
	
			await instance.boot(options);
			await instance.visit('`+url+`', options);
			await info.deferredPromise;
	
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
	err = chromedp.Run(ctx, actions...)
	if err != nil {
		return "", err
	}

	// handle log errors
	if len(logErrors) > 0 {
		return "", fmt.Errorf("log errors: %s", logErrors)
	}

	// write html
	return html, nil
}
