package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/256dpi/ember"
	"github.com/256dpi/ember/fastboot"
)

var name = flag.String("name", "example", "")
var render = flag.Bool("fastboot", false, "")
var isolated = flag.Bool("isolated", false, "")
var addr = flag.String("addr", ":8000", "")
var baseURL = flag.String("base-url", "http://localhost:8000", "")

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
			BaseURL:  *baseURL,
			Isolated: *isolated,
			Reporter: func(err error) {
				fmt.Println("==> Error: " + err.Error())
			},
		})
		if err != nil {
			panic(err)
		}
	}

	// run server
	panic(http.ListenAndServe(*addr, handler))
}
