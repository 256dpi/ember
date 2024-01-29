package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/kr/pretty"

	"github.com/256dpi/ember"
	"github.com/256dpi/ember/fastboot"
)

var name = flag.String("name", "example", "The Ember.js application name.")
var render = flag.Bool("fastboot", false, "Whether to render the application using FastBoot.")
var cache = flag.Duration("cache", 0, "The duration for which to cache rendered pages.")
var isolated = flag.Bool("isolated", false, "Whether to boot the application per request.")
var origin = flag.String("origin", "http://localhost:8000", "The origin of the application.")
var addr = flag.String("addr", ":8000", "The address to listen on.")
var headed = flag.Bool("headed", false, "Whether to run in headed mode (visible Chrome window).")
var log = flag.Bool("log", false, "Whether to log requests and results.")

func main() {
	// parse flags
	flag.Parse()

	// get path
	path, err := filepath.Abs(flag.Arg(0))
	if err != nil {
		panic(err)
	}

	// get directory
	dir := filepath.Base(path)
	path = filepath.Dir(path)

	// prepare files
	files := ember.MustFiles(os.DirFS(path), dir)

	// create app
	app := ember.MustCreate(*name, files)

	// create handler
	var handler http.Handler = app

	// handle fastboot
	if *render {
		handler, err = fastboot.Handle(fastboot.Options{
			App:      app,
			Origin:   *origin,
			Cache:    *cache,
			Isolated: *isolated,
			Headed:   *headed,
			OnRequest: func(request *fastboot.Request) {
				if *log {
					_, _ = pretty.Println("==> Request", request)
				}
			},
			OnResult: func(result *fastboot.Result) {
				if *log {
					_, _ = pretty.Println("==> Result", result)
				}
			},
			OnError: func(err error) {
				_, _ = fmt.Println("==> Error: " + err.Error())
			},
		})
		if err != nil {
			panic(err)
		}
	}

	// run server
	panic(http.ListenAndServe(*addr, handler))
}
