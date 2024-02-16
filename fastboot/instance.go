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
	"time"

	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/log"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"

	"github.com/256dpi/ember"
)

//go:embed script.js
var script string

type manifest struct {
	Fastboot struct {
		Manifest struct {
			AppFiles    []string `json:"appFiles"`
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

// Instance represents a running Fastboot instance.
type Instance struct {
	app    *ember.App
	origin string
	man    manifest
	ctx    context.Context
	cancel func()
	errs   []error
	mutex  sync.Mutex
}

// Boot will boot the provided Fastboot-capable app in a headless browser and
// return a running instance.
func Boot(app *ember.App, origin string, headed bool) (*Instance, error) {
	// trim origin
	origin = strings.TrimRight(origin, "/")

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

	// prepare allocator options
	options := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	if headed {
		options = append(options, chromedp.Flag("headless", false))
	}

	// disable image loading
	options = append(options, chromedp.Flag("blink-settings", "imagesEnabled=false"))

	// create allocator
	ctx, cancel1 := chromedp.NewExecAllocator(context.Background(), options...)

	// create context
	ctx, cancel2 := chromedp.NewContext(ctx)

	// prepare instance
	instance := &Instance{
		app:    app,
		origin: origin,
		man:    manifest,
		ctx:    ctx,
		cancel: func() {
			cancel2()
			cancel1()
		},
	}

	// handle events
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		// intercept main request
		if ev, ok := ev.(*fetch.EventRequestPaused); ok {
			go func() {
				if strings.TrimRight(ev.Request.URL, "/") == origin {
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

		// handle errors
		if ev, ok := ev.(*log.EventEntryAdded); ok {
			if ev.Entry.Level == log.LevelError {
				instance.errs = append(instance.errs, fmt.Errorf("%s (%s)", ev.Entry.Text, ev.Entry.URL))
			}
		}
	})

	// boot instance
	err = instance.boot()
	if err != nil {
		instance.Close()
		return nil, err
	}

	return instance, nil
}

func (i *Instance) boot() error {
	// clone app
	app := i.app.Clone()

	// disable autoboot
	settings := app.Get("APP").(map[string]interface{})
	settings["autoboot"] = false
	app.Set("APP", settings)

	// marshal config
	config, err := json.Marshal(app.Config())
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// prepare actions
	var actions []chromedp.Action

	// enable fetch interception
	actions = append(actions, fetch.Enable())

	// open origin (gets intercepted)
	actions = append(actions, chromedp.Navigate(i.origin))

	// setup environment
	actions = append(actions, chromedp.Evaluate(script, nil))
	actions = append(actions, chromedp.Evaluate(fmt.Sprintf(`$setup("%s", %s)`, i.app.Name(), string(config)), nil, awaitPromise))

	// evaluate scripts
	for _, file := range i.man.Fastboot.Manifest.VendorFiles {
		actions = append(actions, chromedp.Evaluate(string(i.app.File(file)), nil))
	}
	for _, file := range i.man.Fastboot.Manifest.AppFiles {
		actions = append(actions, chromedp.Evaluate(string(i.app.File(file)), nil))
	}

	// boot application
	actions = append(actions, chromedp.Evaluate("$boot()", nil, awaitPromise))

	// run actions
	err = chromedp.Run(i.ctx, actions...)
	if err != nil {
		return fmt.Errorf("failed to boot application: %w", err)
	}

	return nil
}

// Visit will visit the provided URL and return the result.
func (i *Instance) Visit(url string, r Request, timeout time.Duration) (Result, error) {
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

	// render application
	actions = append(actions, chromedp.Evaluate(fmt.Sprintf(`$render("%s", %s)`, url, string(data)), nil, awaitPromise))

	// capture HTML
	var result Result
	actions = append(actions, chromedp.Evaluate(`$capture()`, &result))

	// ensure timeout
	ctx, cancel := context.WithTimeout(i.ctx, timeout)
	defer cancel()

	// run actions
	err = chromedp.Run(ctx, actions...)
	if err != nil {
		// reboot instance on timeout
		if errors.Is(err, context.DeadlineExceeded) {
			_ = i.boot()
		}

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
func Render(app *ember.App, location string, r Request, timeout time.Duration) (Result, error) {
	// parse URL
	urlData, err := url.Parse(location)
	if err != nil {
		return Result{}, err
	}

	// determine origin
	origin := fmt.Sprintf("%s://%s", urlData.Scheme, urlData.Host)

	// determine visit URL
	urlData.Scheme = ""
	urlData.Opaque = ""
	urlData.Host = ""
	visitURL := urlData.String()

	// boot app
	instance, err := Boot(app, origin, false)
	if err != nil {
		return Result{}, err
	}
	defer instance.Close()

	// visit URL
	result, err := instance.Visit(visitURL, r, timeout)
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

func awaitPromise(params *runtime.EvaluateParams) *runtime.EvaluateParams {
	params.AwaitPromise = true
	return params
}
