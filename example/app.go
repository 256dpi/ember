package example

import (
	"embed"

	"github.com/256dpi/ember"
)

//go:embed dist
var files embed.FS

// App creates and returns the app handler.
func App() *ember.App {
	return ember.MustCreate("example", ember.MustFiles(files, "dist"))
}
