package ember

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApp(t *testing.T) {
	app, err := Create("app", map[string]string{
		"index.html": indexHTML,
		"script.js":  `alert("Hello World!");`,
	})
	assert.NoError(t, err)

	assert.Equal(t, nil, app.Get("foo"))

	app.Set("foo", map[string]interface{}{
		"bar": 3.14,
		"baz": "quz qux",
	})

	assert.Equal(t, map[string]interface{}{
		"bar": 3.14,
		"baz": "quz qux",
	}, app.Get("foo"))

	app.AddFile("foo.html", "Hello World!")
	app.AddInlineStyle("body { background: red; }")
	app.AddInlineScript(`alert("Hello World!);"`)

	assert.False(t, app.IsAsset(""))
	assert.False(t, app.IsAsset("foo"))
	assert.False(t, app.IsAsset("/index.html"))
	assert.True(t, app.IsAsset("/foo.html"))
	assert.True(t, app.IsAsset("/script.js"))

	assert.True(t, app.IsPage(""))
	assert.True(t, app.IsPage("foo"))
	assert.True(t, app.IsPage("/index.html"))
	assert.False(t, app.IsPage("/foo.html"))
	assert.False(t, app.IsPage("/script.js"))

	index := app.files["index.html"]
	assert.Equal(t, unIndent(`<!DOCTYPE html>
		<html>
			<head>
				<meta charset="utf-8"/>
				<meta http-equiv="X-UA-Compatible" content="IE=edge"/>
				<title>App</title>
				<meta name="description" content=""/>
				<meta name="viewport" content="width=device-width, initial-scale=1"/>
				<meta name="app/config/environment" content="%7B%22APP%22:%7B%22name%22:%22app%22%2C%22version%22:%220.0.0+a7250a80%22%7D%2C%22EmberENV%22:%7B%22EXTEND_PROTOTYPES%22:%7B%22Date%22:false%7D%2C%22FEATURES%22:%7B%7D%2C%22_JQUERY_INTEGRATION%22:false%2C%22_TEMPLATE_ONLY_GLIMMER_COMPONENTS%22:true%7D%2C%22environment%22:%22production%22%2C%22exportApplicationGlobal%22:false%2C%22foo%22:%7B%22bar%22:3.14%2C%22baz%22:%22quz%20qux%22%7D%2C%22locationType%22:%22auto%22%2C%22modulePrefix%22:%22app%22%2C%22rootURL%22:%22%2F%22%7D"/>
				<link integrity="" rel="stylesheet" href="/assets/vendor-d41d8cd98f00b204e9800998ecf8427e.css"/>
				<link integrity="" rel="stylesheet" href="/assets/app-45c749a3bbece8e3ce4ffd9e6b8addf7.css"/>
				<style>body { background: red; }</style>
			</head>
			<body>
				<script src="/assets/vendor-0602240bb8c898070836851c4cc335bd.js" integrity="sha256-x5KZQsQtD11ZTdqNAQIXsfX2GhhsgLLMP2D6P/QUXtc= sha512-JeMuQGObr+XCFa0pndQDId4cKiqROg4Ai0iR27Zgv9FE32p340XLGz6OpQm8PrmcRGShcxPNkh61sc19Sm87Lw=="></script>
				<script src="/assets/app-6a49fc3c244bed354719f50d3ca3dd38.js" integrity="sha256-Tf7uETTbqK91hJxzmSrymkqPCl8zrt7KEnQ46H7MlSo= sha512-/G/3aD3HMrxRYLK4mUFz7Cbo3miN0lKYHrknOFSzwqop4LOcVMSc02FpvKJFWUm91Ga0DvgC3wN4I4RboTBfLQ=="></script>
				<script>alert("Hello World!);"</script>
			</body>
		</html>
	`), unIndent(string(index)))

	foo := app.files["foo.html"]
	assert.Equal(t, "Hello World!", string(foo))

	app.PrefixAssetsPaths("foo")
	index = app.files["index.html"]
	assert.Equal(t, unIndent(`<!DOCTYPE html>
		<html>
			<head>
				<meta charset="utf-8"/>
				<meta http-equiv="X-UA-Compatible" content="IE=edge"/>
				<title>App</title>
				<meta name="description" content=""/>
				<meta name="viewport" content="width=device-width, initial-scale=1"/>
				<meta name="app/config/environment" content="%7B%22APP%22:%7B%22name%22:%22app%22%2C%22version%22:%220.0.0+a7250a80%22%7D%2C%22EmberENV%22:%7B%22EXTEND_PROTOTYPES%22:%7B%22Date%22:false%7D%2C%22FEATURES%22:%7B%7D%2C%22_JQUERY_INTEGRATION%22:false%2C%22_TEMPLATE_ONLY_GLIMMER_COMPONENTS%22:true%7D%2C%22environment%22:%22production%22%2C%22exportApplicationGlobal%22:false%2C%22foo%22:%7B%22bar%22:3.14%2C%22baz%22:%22quz%20qux%22%7D%2C%22locationType%22:%22auto%22%2C%22modulePrefix%22:%22app%22%2C%22rootURL%22:%22%2F%22%7D"/>
				<link integrity="" rel="stylesheet" href="/foo/assets/vendor-d41d8cd98f00b204e9800998ecf8427e.css"/>
				<link integrity="" rel="stylesheet" href="/foo/assets/app-45c749a3bbece8e3ce4ffd9e6b8addf7.css"/>
				<style>body { background: red; }</style>
			</head>
			<body>
				<script src="/foo/assets/vendor-0602240bb8c898070836851c4cc335bd.js" integrity="sha256-x5KZQsQtD11ZTdqNAQIXsfX2GhhsgLLMP2D6P/QUXtc= sha512-JeMuQGObr+XCFa0pndQDId4cKiqROg4Ai0iR27Zgv9FE32p340XLGz6OpQm8PrmcRGShcxPNkh61sc19Sm87Lw=="></script>
				<script src="/foo/assets/app-6a49fc3c244bed354719f50d3ca3dd38.js" integrity="sha256-Tf7uETTbqK91hJxzmSrymkqPCl8zrt7KEnQ46H7MlSo= sha512-/G/3aD3HMrxRYLK4mUFz7Cbo3miN0lKYHrknOFSzwqop4LOcVMSc02FpvKJFWUm91Ga0DvgC3wN4I4RboTBfLQ=="></script>
				<script>alert("Hello World!);"</script>
			</body>
		</html>
	`), unIndent(string(index)))
}

func BenchmarkAppCloneSet(b *testing.B) {
	app := MustCreate("app", files)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		app.Clone().Set("foo", "bar")
	}
}
