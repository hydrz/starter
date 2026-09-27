package httpserver

import (
	"encoding/json"
	"net/http"

	"gopkg.in/yaml.v3"
)

var openAPIErr error

func loadOpenAPISpecs() error {
	openAPIOnce.Do(func() {
		openAPIYAML, openAPIErr = docsAssets.ReadFile("assets/openapi.yaml")
		if openAPIErr != nil {
			return
		}
		var raw any
		if openAPIErr = yaml.Unmarshal(openAPIYAML, &raw); openAPIErr != nil {
			return
		}
		openAPIJSON, openAPIErr = json.Marshal(raw)
	})
	return openAPIErr
}

func openAPISpecYAML(response http.ResponseWriter, _ *http.Request) {
	if err := loadOpenAPISpecs(); err != nil {
		http.Error(response, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	response.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	_, _ = response.Write(openAPIYAML)
}

func openAPISpecJSON(response http.ResponseWriter, _ *http.Request) {
	if err := loadOpenAPISpecs(); err != nil {
		http.Error(response, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = response.Write(openAPIJSON)
}

func scalarReference(response http.ResponseWriter, _ *http.Request) {
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	response.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src data: https:; font-src data:; connect-src 'self'")
	_, _ = response.Write([]byte(scalarHTML))
}

func scalarScript(response http.ResponseWriter, _ *http.Request) {
	script, err := docsAssets.ReadFile("assets/scalar.js")
	if err != nil {
		http.Error(response, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	response.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	response.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = response.Write(script)
}

const scalarHTML = `<!doctype html>
<html lang="zh-CN">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>Starter API</title>
  </head>
  <body>
    <script id="api-reference" data-url="/api/openapi.yaml"></script>
    <script src="/api/docs/scalar.js"></script>
  </body>
</html>`
