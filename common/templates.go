package common

import (
	"fmt"
	html "html/template"
	"net/http"

	fs "github.com/zorchenhimer/MovieNight/files"
)

// Holds the server's templates
var serverTemplates map[string]*html.Template

// Called from the server
func InitTemplates() error {
	serverTemplates = make(map[string]*html.Template)

	// keys and files to load for that template
	var serverTemplateDefs map[string][]string = map[string][]string{
		"pin":    {"static/base.html", "static/thedoor.html"},
		"main":   {"static/base.html", "static/main.html"},
		"help":   {"static/base.html", "static/help.html"},
		"emotes": {"static/base.html", "static/emotes.html"},
	}

	// Parse server templates
	for key, files := range serverTemplateDefs {
		t, err := html.ParseFS(fs.StaticFS, files...)
		if err != nil {
			return fmt.Errorf("unable to parse templates for %s: %w", key, err)
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
