package fastboot

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/256dpi/ember/example"
)

func TestFastboot(t *testing.T) {
	app := example.App()

	html, err := Visit(app, "/")

	assert.NoError(t, err)
	assert.Contains(t, html, "<h1>Example</h1>")
	assert.Contains(t, html, "Is FastBoot: true")
}

func TestFastbootGitHub(t *testing.T) {
	app := example.App()

	html, err := Visit(app, "/github")

	assert.NoError(t, err)
	assert.Contains(t, html, "<h1>Example</h1>")
	assert.Contains(t, html, "Name: Joël Gähwiler")
}

func BenchmarkFastboot(b *testing.B) {
	app := example.App()

	for i := 0; i < b.N; i++ {
		html, err := Visit(app, "/")
		assert.NoError(b, err)
		assert.NotZero(b, html)
	}
}
