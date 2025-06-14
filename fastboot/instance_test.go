package fastboot

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/256dpi/ember/example"
)

const timeout = 5 * time.Second

func TestRender(t *testing.T) {
	app := example.App()

	result, err := Render(app, "https://example.org/", Request{Path: "/"}, timeout)
	assert.NoError(t, err)
	assert.Contains(t, result.HTML(), "<h1>Example</h1>")
	assert.Contains(t, result.HTML(), "<p>Is FastBoot: true</p>")

	result, err = Render(app, "https://example.org/delay?timeout=500", Request{Path: "/delay"}, timeout)
	assert.NoError(t, err)
	assert.Contains(t, result.HTML(), "<h1>Example</h1>")
	assert.Contains(t, result.HTML(), "<p>Message: Hello world!</p>")

	result, err = Render(app, "https://example.org/fetch", Request{Path: "/fetch"}, timeout)
	assert.NoError(t, err)
	assert.Contains(t, result.HTML(), "<h1>Example</h1>")
	assert.Contains(t, result.HTML(), "<p>Name: Joël Gähwiler</p>")
}

func TestRenderResult(t *testing.T) {
	app := example.App()

	result, err := Render(app, "https://example.org/?attributes=1", Request{
		Path: "/",
		QueryParams: map[string]string{
			"attributes": "1",
		}}, time.Second)
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

	result, err = Render(app, "https://example.org/", Request{Path: "/"}, timeout)
	assert.NoError(t, err)
	assert.Equal(t, Result{
		HeadContent:    "<title>Example</title>",
		BodyContent:    "\n\n<h1>Example</h1>\n\n<p>Is FastBoot: true</p>",
		HTMLAttributes: map[string]string{},
		HeadAttributes: map[string]string{},
		BodyAttributes: map[string]string{},
	}, result)
}

func TestRenderDebug(t *testing.T) {
	app := example.App()

	result, err := Render(app, "https://example.org/debug", Request{
		Method:   "GET",
		Protocol: "http:",
		Path:     "/debug",
		Headers: map[string][]string{
			"Accept": {"text/html"},
			"Host":   {"example.org"},
		},
		Cookies: map[string]string{
			"foo": "bar",
		},
		QueryParams: map[string]string{
			"bar": "baz",
		},
		Body: "quz",
	}, timeout)
	assert.NoError(t, err)

	_, raw, _ := strings.Cut(result.BodyContent, "</h1>")
	var out map[string]interface{}
	err = json.Unmarshal([]byte(raw), &out)
	assert.NoError(t, err)
	assert.Equal(t, map[string]interface{}{
		"isFastBoot": true,
		"request": map[string]interface{}{
			"method":   "GET",
			"protocol": "http:",
			"path":     "/debug",
			"headers": map[string]interface{}{
				"headers": map[string]interface{}{
					"accept": []interface{}{"text/html"},
					"host":   []interface{}{"example.org"},
				},
			},
			"cookies": map[string]interface{}{
				"foo": "bar",
			},
			"queryParams": map[string]interface{}{
				"bar": "baz",
			},
			"body": "quz",
		},
		"requestHost": "example.org",
		"response": map[string]interface{}{
			"statusCode": 200.0,
			"headers": map[string]interface{}{
				"headers": map[string]interface{}{},
			},
		},
		"metadata": map[string]interface{}{},
	}, out)
}

func TestInstance(t *testing.T) {
	app := example.App()

	instance, err := Boot(app, "https://example.org", false)
	assert.NoError(t, err)
	defer instance.Close()

	result, err := instance.Visit("/", Request{Path: "/"}, timeout)
	assert.NoError(t, err)
	assert.Contains(t, result.HTML(), "<h1>Example</h1>")
	assert.Contains(t, result.HTML(), "<p>Is FastBoot: true</p>")

	result, err = instance.Visit("/delay?timeout=500", Request{Path: "/delay"}, timeout)
	assert.NoError(t, err)
	assert.Contains(t, result.HTML(), "<h1>Example</h1>")
	assert.Contains(t, result.HTML(), "<p>Message: Hello world!</p>")

	result, err = instance.Visit("/fetch", Request{Path: "/fetch"}, timeout)
	assert.NoError(t, err)
	assert.Contains(t, result.HTML(), "<h1>Example</h1>")
	assert.Contains(t, result.HTML(), "<p>Name: Joël Gähwiler</p>")
}

func TestInstanceTimeout(t *testing.T) {
	app := example.App()

	instance, err := Boot(app, "https://example.org", false)
	assert.NoError(t, err)
	defer instance.Close()

	result, err := instance.Visit("/delay?timeout=5000", Request{Path: "/delay"}, timeout)
	assert.Error(t, err)
	assert.Empty(t, result)

	result, err = instance.Visit("/delay?timeout=500", Request{Path: "/delay"}, timeout)
	assert.NoError(t, err)
	assert.Contains(t, result.HTML(), "<h1>Example</h1>")
	assert.Contains(t, result.HTML(), "<p>Message: Hello world!</p>")
}

func BenchmarkRender(b *testing.B) {
	app := example.App()

	for i := 0; i < b.N; i++ {
		result, err := Render(app, "https://example.org/", Request{Path: "/"}, timeout)
		assert.NoError(b, err)
		assert.NotZero(b, result.HTML())
	}
}

func BenchmarkInstance(b *testing.B) {
	app := example.App()

	instance, err := Boot(app, "https://example.org", false)
	assert.NoError(b, err)
	defer instance.Close()

	for i := 0; i < b.N; i++ {
		html, err := instance.Visit("/", Request{Path: "/"}, timeout)
		assert.NoError(b, err)
		assert.NotZero(b, html)
	}
}
