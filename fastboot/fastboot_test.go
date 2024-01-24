package fastboot

import (
	"context"
	"testing"

	"github.com/chromedp/chromedp"
	"github.com/stretchr/testify/assert"

	"github.com/256dpi/ember/example"
)

func TestFastboot(t *testing.T) {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	app := example.App()

	html, err := Visit(ctx, app, "/")
	assert.NoError(t, err)
	assert.Contains(t, html, "<h1>Example</h1>")
	assert.Contains(t, html, "Is FastBoot: true")
}

func TestFastbootGitHub(t *testing.T) {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	app := example.App()

	html, err := Visit(ctx, app, "/github")

	assert.NoError(t, err)
	assert.Contains(t, html, "<h1>Example</h1>")
	assert.Contains(t, html, "Name: Joël Gähwiler")
}

func BenchmarkFastboot(b *testing.B) {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	app := example.App()

	for i := 0; i < b.N; i++ {
		html, err := Visit(ctx, app, "/")
		assert.NoError(b, err)
		assert.NotZero(b, html)
	}
}
