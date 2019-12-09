package ember

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

const indexHTML = `<!DOCTYPE html>
<html>
  	<head>
		<meta charset="utf-8"/>
		<meta http-equiv="X-UA-Compatible" content="IE=edge"/>
		<title>App</title>
		<meta name="description" content=""/>
		<meta name="viewport" content="width=device-width, initial-scale=1"/>
		<meta name="app/config/environment" content="%7B%22modulePrefix%22%3A%22app%22%2C%22environment%22%3A%22production%22%2C%22rootURL%22%3A%22%2F%22%2C%22locationType%22%3A%22auto%22%2C%22EmberENV%22%3A%7B%22FEATURES%22%3A%7B%7D%2C%22EXTEND_PROTOTYPES%22%3A%7B%22Date%22%3Afalse%7D%2C%22_JQUERY_INTEGRATION%22%3Afalse%2C%22_TEMPLATE_ONLY_GLIMMER_COMPONENTS%22%3Atrue%7D%2C%22APP%22%3A%7B%22name%22%3A%22app%22%2C%22version%22%3A%220.0.0%2Ba7250a80%22%7D%2C%22exportApplicationGlobal%22%3Afalse%7D"/>
		<link integrity="" rel="stylesheet" href="/assets/vendor-d41d8cd98f00b204e9800998ecf8427e.css"/>
		<link integrity="" rel="stylesheet" href="/assets/app-45c749a3bbece8e3ce4ffd9e6b8addf7.css"/>
  	</head>
  	<body>
		<script src="/assets/vendor-0602240bb8c898070836851c4cc335bd.js" integrity="sha256-x5KZQsQtD11ZTdqNAQIXsfX2GhhsgLLMP2D6P/QUXtc= sha512-JeMuQGObr+XCFa0pndQDId4cKiqROg4Ai0iR27Zgv9FE32p340XLGz6OpQm8PrmcRGShcxPNkh61sc19Sm87Lw=="></script>
		<script src="/assets/app-6a49fc3c244bed354719f50d3ca3dd38.js" integrity="sha256-Tf7uETTbqK91hJxzmSrymkqPCl8zrt7KEnQ46H7MlSo= sha512-/G/3aD3HMrxRYLK4mUFz7Cbo3miN0lKYHrknOFSzwqop4LOcVMSc02FpvKJFWUm91Ga0DvgC3wN4I4RboTBfLQ=="></script>
		<div id="ember-basic-dropdown-wormhole"></div>
  	</body>
</html>`

var files = map[string]string{
	"index.html": indexHTML,
}

func Example() {
	// create app
	app, err := Create("app", files)
	if err != nil {
		panic(err)
	}

	// set static config
	err = app.Set("apiBaseURI", "http://api.example.com")
	if err != nil {
		panic(err)
	}

	// run listener
	go func() {
		panic(http.ListenAndServe("0.0.0.0:4242", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// clone app
			app := app.Clone()

			// set dynamic config
			err = app.Set("path", r.URL.Path)
			if err != nil {
				panic(err)
			}

			// serve app
			app.ServeHTTP(w, r)
		})))
	}()
	time.Sleep(10 * time.Millisecond)

	// get page
	res, err := http.Get("http://0.0.0.0:4242/hello")
	if err != nil {
		panic(err)
	}

	// ensure body is closed
	defer res.Body.Close()

	// read body
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	fmt.Println(clean(string(data)))

	// Output:
	// <!DOCTYPE html><html><head>
	// <meta charset="utf-8"/>
	// <meta http-equiv="X-UA-Compatible" content="IE=edge"/>
	// <title>App
	// </title>
	// <meta name="description" content=""/>
	// <meta name="viewport" content="width=device-width, initial-scale=1"/>
	// <meta name="app/config/environment" content="%7B%22APP%22%3A%7B%22name%22%3A%22app%22%2C%22version%22%3A%220.0.0%2Ba7250a80%22%7D%2C%22EmberENV%22%3A%7B%22EXTEND_PROTOTYPES%22%3A%7B%22Date%22%3Afalse%7D%2C%22FEATURES%22%3A%7B%7D%2C%22_JQUERY_INTEGRATION%22%3Afalse%2C%22_TEMPLATE_ONLY_GLIMMER_COMPONENTS%22%3Atrue%7D%2C%22apiBaseURI%22%3A%22http%3A%2F%2Fapi.example.com%22%2C%22environment%22%3A%22production%22%2C%22exportApplicationGlobal%22%3Afalse%2C%22locationType%22%3A%22auto%22%2C%22modulePrefix%22%3A%22app%22%2C%22path%22%3A%22%2Fhello%22%2C%22rootURL%22%3A%22%2F%22%7D"/>
	// <link integrity="" rel="stylesheet" href="/assets/vendor-d41d8cd98f00b204e9800998ecf8427e.css"/>
	// <link integrity="" rel="stylesheet" href="/assets/app-45c749a3bbece8e3ce4ffd9e6b8addf7.css"/>
	// </head>
	// <body>
	// <script src="/assets/vendor-0602240bb8c898070836851c4cc335bd.js" integrity="sha256-x5KZQsQtD11ZTdqNAQIXsfX2GhhsgLLMP2D6P/QUXtc= sha512-JeMuQGObr+XCFa0pndQDId4cKiqROg4Ai0iR27Zgv9FE32p340XLGz6OpQm8PrmcRGShcxPNkh61sc19Sm87Lw=="></script>
	// <script src="/assets/app-6a49fc3c244bed354719f50d3ca3dd38.js" integrity="sha256-Tf7uETTbqK91hJxzmSrymkqPCl8zrt7KEnQ46H7MlSo= sha512-/G/3aD3HMrxRYLK4mUFz7Cbo3miN0lKYHrknOFSzwqop4LOcVMSc02FpvKJFWUm91Ga0DvgC3wN4I4RboTBfLQ=="></script>
	// <div id="ember-basic-dropdown-wormhole"></div>
	// </body></html>
}
