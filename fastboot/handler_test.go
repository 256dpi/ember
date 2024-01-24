package fastboot

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/256dpi/ember/example"
)

func TestHandler(t *testing.T) {
	app := example.App()

	handler, err := Handle(app)
	assert.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/?attributes=1", nil)
	handler.ServeHTTP(rec, req)
	assert.Equal(t, `<!DOCTYPE html>
<html foo="html">
  <head foo="head">
    <meta charset="utf-8" />
    <meta name="description" content="" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />

    
<meta name="example/config/environment" content="%7B%22modulePrefix%22%3A%22example%22%2C%22environment%22%3A%22production%22%2C%22rootURL%22%3A%22%2F%22%2C%22locationType%22%3A%22history%22%2C%22EmberENV%22%3A%7B%22EXTEND_PROTOTYPES%22%3Afalse%2C%22FEATURES%22%3A%7B%7D%2C%22_APPLICATION_TEMPLATE_WRAPPER%22%3Afalse%2C%22_DEFAULT_ASYNC_OBSERVERS%22%3Atrue%2C%22_JQUERY_INTEGRATION%22%3Afalse%2C%22_TEMPLATE_ONLY_GLIMMER_COMPONENTS%22%3Atrue%7D%2C%22APP%22%3A%7B%22name%22%3A%22example%22%2C%22version%22%3A%220.0.0%2Befcaa952%22%7D%7D" />
<!-- EMBER_CLI_FASTBOOT_TITLE --><title>Example</title>

    <link integrity="" rel="stylesheet" href="/assets/vendor-d41d8cd98f00b204e9800998ecf8427e.css" />
    <link integrity="" rel="stylesheet" href="/assets/example-d41d8cd98f00b204e9800998ecf8427e.css" />

    
  </head>
  <body foo="body">
    

<h1>Example</h1>

<p>Is FastBoot: true</p>

    <script src="/assets/vendor-98e4d70515266258f27b271e42403a97.js" integrity="sha256-VHHb4zGGwB7uYxBRR0vGwZRXppElnCGg4QMLN4V4x3g= sha512-R0bbeKQXIW+sFq59Ta9ZTwt9TAsSTzc6oX0Ssd1mLddVdVCX5U8wEnu7+Zw0aLLxHTVAF8juBqslZ2jMevY8cw==" ></script>
    <script src="/assets/example-5a294e723652b7e4bce6c248b76fc996.js" integrity="sha256-IDY7PfIO2lrlzXPORItOlV1w0C0x7uK5VXGVXjVzaug= sha512-YURw6MjgaWNdhctmEJNkYfvTco9Oh8dfYcjQnwB9GyiNWcuUtVPilfgpRhnT0EWLg+pRnGwX3CmAmYRXGxcSSg==" ></script>

    
  </body>
</html>`, strings.TrimSpace(rec.Body.String()))

	rec = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/index.html", nil)
	handler.ServeHTTP(rec, req)
	assert.Equal(t, string(app.File("index.html")), rec.Body.String())
}
