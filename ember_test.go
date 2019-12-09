package serve

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApp(t *testing.T) {
	app, err := Create("app", map[string]string{
		"index.html": indexHTML,
	})
	assert.NoError(t, err)

	err = app.Set("foo", "bar")
	assert.NoError(t, err)

	index := app.files["index.html"]
	assert.Equal(t, clean(`<!DOCTYPE html>
		<html>
			<head>
				<meta charset="utf-8"/>
				<meta http-equiv="X-UA-Compatible" content="IE=edge"/>
				<title>App</title>
				<meta name="description" content=""/>
				<meta name="viewport" content="width=device-width, initial-scale=1"/>
				<meta name="app/config/environment" content="%7B%22APP%22%3A%7B%22name%22%3A%22app%22%2C%22version%22%3A%220.0.0%2Ba7250a80%22%7D%2C%22EmberENV%22%3A%7B%22EXTEND_PROTOTYPES%22%3A%7B%22Date%22%3Afalse%7D%2C%22FEATURES%22%3A%7B%7D%2C%22_JQUERY_INTEGRATION%22%3Afalse%2C%22_TEMPLATE_ONLY_GLIMMER_COMPONENTS%22%3Atrue%7D%2C%22environment%22%3A%22production%22%2C%22exportApplicationGlobal%22%3Afalse%2C%22foo%22%3A%22bar%22%2C%22locationType%22%3A%22auto%22%2C%22modulePrefix%22%3A%22app%22%2C%22rootURL%22%3A%22%2F%22%7D"/>
				<link integrity="" rel="stylesheet" href="/assets/vendor-d41d8cd98f00b204e9800998ecf8427e.css"/>
				<link integrity="" rel="stylesheet" href="/assets/app-45c749a3bbece8e3ce4ffd9e6b8addf7.css"/>
			</head>
			<body>
				<script src="/assets/vendor-0602240bb8c898070836851c4cc335bd.js" integrity="sha256-x5KZQsQtD11ZTdqNAQIXsfX2GhhsgLLMP2D6P/QUXtc= sha512-JeMuQGObr+XCFa0pndQDId4cKiqROg4Ai0iR27Zgv9FE32p340XLGz6OpQm8PrmcRGShcxPNkh61sc19Sm87Lw=="></script>
				<script src="/assets/app-6a49fc3c244bed354719f50d3ca3dd38.js" integrity="sha256-Tf7uETTbqK91hJxzmSrymkqPCl8zrt7KEnQ46H7MlSo= sha512-/G/3aD3HMrxRYLK4mUFz7Cbo3miN0lKYHrknOFSzwqop4LOcVMSc02FpvKJFWUm91Ga0DvgC3wN4I4RboTBfLQ=="></script>
				<div id="ember-basic-dropdown-wormhole"></div>
			</body>
		</html>
	`), clean(string(index)))
}
