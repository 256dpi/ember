package fastboot

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/256dpi/ember/example"
)

func TestHandler(t *testing.T) {
	app := example.App()

	handler, err := Handle(Options{
		App:    app,
		Origin: "https://example.org",
		OnError: func(err error) {
			assert.NoError(t, err)
		},
	})
	assert.NoError(t, err)
	defer handler.Close()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "https://example.org/?attributes=1", nil)
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	assert.Contains(t, body, `<html foo="html">`)
	assert.Contains(t, body, `<head foo="head">`)
	assert.Contains(t, body, `<title>Example</title>`)
	assert.Contains(t, body, `<body foo="body">`)
	assert.Contains(t, body, `<h1>Example</h1>`)
	assert.Contains(t, body, `<p>Is FastBoot: true</p>`)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "https://example.org/index.html", nil)
	handler.ServeHTTP(rec, req)
	assert.Equal(t, string(app.File("index.html")), rec.Body.String())
}

func TestHandlerIsolated(t *testing.T) {
	app := example.App()

	handler, err := Handle(Options{
		App:      app,
		Origin:   "https://example.org",
		Isolated: true,
		OnError: func(err error) {
			assert.NoError(t, err)
		},
	})
	assert.NoError(t, err)
	defer handler.Close()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "https://example.org/?attributes=1", nil)
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	assert.Contains(t, body, `<html foo="html">`)
	assert.Contains(t, body, `<head foo="head">`)
	assert.Contains(t, body, `<title>Example</title>`)
	assert.Contains(t, body, `<body foo="body">`)
	assert.Contains(t, body, `<h1>Example</h1>`)
	assert.Contains(t, body, `<p>Is FastBoot: true</p>`)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "https://example.org/index.html", nil)
	handler.ServeHTTP(rec, req)
	assert.Equal(t, string(app.File("index.html")), rec.Body.String())
}

func BenchmarkHandlerCache(b *testing.B) {
	app := example.App()

	handler, err := Handle(Options{
		App:    app,
		Origin: "https://example.org",
		Cache:  time.Second,
		OnError: func(err error) {
			assert.NoError(b, err)
		},
	})
	assert.NoError(b, err)
	defer handler.Close()

	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "https://example.org/?attributes=1", nil)
		handler.ServeHTTP(rec, req)
		assert.NotZero(b, rec.Body.Len())
	}
}
