package fastboot

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/256dpi/ember/example"
)

func TestRenderBasic(t *testing.T) {
	app := example.App()

	html, err := Render(app, "/")
	assert.NoError(t, err)
	assert.Contains(t, html, "<h1>Example</h1>")
	assert.Contains(t, html, "<p>Is FastBoot: true</p>")
}

func TestRenderDelay(t *testing.T) {
	app := example.App()

	html, err := Render(app, "/delay")

	assert.NoError(t, err)
	assert.Contains(t, html, "<h1>Example</h1>")
	assert.Contains(t, html, "<p>Message: Hello world!</p>")
}

func TestRenderGitHub(t *testing.T) {
	app := example.App()

	html, err := Render(app, "/github")

	assert.NoError(t, err)
	assert.Contains(t, html, "<h1>Example</h1>")
	assert.Contains(t, html, "<p>Name: Joël Gähwiler</p>")
}

func BenchmarkInstance(b *testing.B) {
	app := example.App()

	instance, err := Boot(app)
	assert.NoError(b, err)

	defer instance.Close()

	for i := 0; i < b.N; i++ {
		html, err := instance.Visit("/")
		assert.NoError(b, err)
		assert.NotZero(b, html)
	}
}
