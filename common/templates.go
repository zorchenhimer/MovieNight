package common

import (
	"fmt"
	html "html/template"
	"net/http"
)

// Holds the server's templates
var serverTemplates map[string]*html.Template

// Called from the server
func InitTemplates() error {
	serverTemplates = make(map[string]*html.Template)
	var runPath string = RunPath()

	// keys and files to load for that template
	var serverTemplateDefs map[string][]string = map[string][]string{
		"pin":    {runPath + "/static/base.html", runPath + "/static/thedoor.html"},
		"main":   {runPath + "/static/base.html", runPath + "/static/main.html"},
		"help":   {runPath + "/static/base.html", runPath + "/static/help.html"},
		"emotes": {runPath + "/static/base.html", runPath + "/static/emotes.html"},
	}

	// Parse server templates
	for key, files := range serverTemplateDefs {
		t, err := html.ParseFiles(files...)
		if err != nil {
			return fmt.Errorf("unable to parse templates for %s: %v", key, err)
		}

		serverTemplates[key] = t
	}

	return nil
}

func ExecuteServerTemplate(w http.ResponseWriter, key string, data interface{}) error {
	t, ok := serverTemplates[key]
	if !ok {
		return fmt.Errorf("template with the key %q does not exist", key)
	}

	return t.Execute(w, data)
}
