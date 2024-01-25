package fastboot

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/256dpi/ember/example"
)

func TestHandler(t *testing.T) {
	app := example.App()

	handler, err := Handle(app, "https://example.org", func(err error) {
		assert.NoError(t, err)
	})
	assert.NoError(t, err)

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
