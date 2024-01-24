package fastboot

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/256dpi/ember/example"
)

func TestRender(t *testing.T) {
	app := example.App()

	result, err := Render(app, "/")
	assert.NoError(t, err)
	assert.Contains(t, result.HTML(), "<h1>Example</h1>")
	assert.Contains(t, result.HTML(), "<p>Is FastBoot: true</p>")

	result, err = Render(app, "/delay")
	assert.NoError(t, err)
	assert.Contains(t, result.HTML(), "<h1>Example</h1>")
	assert.Contains(t, result.HTML(), "<p>Message: Hello world!</p>")

	result, err = Render(app, "/github")
	assert.NoError(t, err)
	assert.Contains(t, result.HTML(), "<h1>Example</h1>")
	assert.Contains(t, result.HTML(), "<p>Name: Joël Gähwiler</p>")
}

func TestRenderResult(t *testing.T) {
	app := example.App()

	result, err := Render(app, "/?attributes=1")
	assert.NoError(t, err)
	assert.Equal(t, Result{
		HeadContent: "<title>Example</title>",
		BodyContent: "\n\n<h1>Example</h1>\n\n<p>Is FastBoot: true</p>",
		HTMLAttributes: map[string]string{
			"foo": "html",
		},
		HeadAttributes: map[string]string{
			"foo": "head",
		},
		BodyAttributes: map[string]string{
			"foo": "body",
		},
	}, result)

	result, err = Render(app, "/")
	assert.NoError(t, err)
	assert.Equal(t, Result{
		HeadContent:    "<title>Example</title>",
		BodyContent:    "\n\n<h1>Example</h1>\n\n<p>Is FastBoot: true</p>",
		HTMLAttributes: map[string]string{},
		HeadAttributes: map[string]string{},
		BodyAttributes: map[string]string{},
	}, result)
}

func TestInstance(t *testing.T) {
	app := example.App()

	instance, err := Boot(app)
	assert.NoError(t, err)
	defer instance.Close()

	result, err := instance.Visit("/")
	assert.NoError(t, err)
	assert.Contains(t, result.HTML(), "<h1>Example</h1>")
	assert.Contains(t, result.HTML(), "<p>Is FastBoot: true</p>")

	result, err = instance.Visit("/delay")
	assert.NoError(t, err)
	assert.Contains(t, result.HTML(), "<h1>Example</h1>")
	assert.Contains(t, result.HTML(), "<p>Message: Hello world!</p>")

	result, err = instance.Visit("/github")
	assert.NoError(t, err)
	assert.Contains(t, result.HTML(), "<h1>Example</h1>")
	assert.Contains(t, result.HTML(), "<p>Name: Joël Gähwiler</p>")
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
