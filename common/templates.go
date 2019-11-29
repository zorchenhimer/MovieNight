package common

import (
	"fmt"
	html "html/template"
	"net/http"
	"strings"
	text "text/template"
)

// Holds the server's templates
var serverTemplates map[string]*html.Template

// Holds the client's chat templates
var chatTemplates map[string]*text.Template

var isServer bool = false

// keys and files to load for that template
var serverTemplateDefs map[string][]string = map[string][]string{
	"pin":    []string{"./static/base.html", "./static/thedoor.html"},
	"main":   []string{"./static/base.html", "./static/main.html"},
	"help":   []string{"./static/base.html", "./static/help.html"},
	"emotes": []string{"./static/base.html", "./static/emotes.html"},
}

var chatTemplateDefs map[string]string = map[string]string{
	fmt.Sprint(DTInvalid, 0): "wot",

	fmt.Sprint(DTChat, MsgChat): `<span>{{.Badge}} <span class="name" style="color:{{.Color}}">{{.From}}` +
		`</span><b>:</b> <span class="msg">{{.Message}}</span></span>`,
	fmt.Sprint(DTChat, MsgAction): `<span style="color:{{.Color}}"><span class="name">{{.From}}` +
		`</span> <span class="cmdme">{{.Message}}</span></span>`,
}

// Called from the server
func InitTemplates() error {
	isServer = true
	serverTemplates = make(map[string]*html.Template)
	chatTemplates = make(map[string]*text.Template)

	// Parse server templates
	for key, files := range serverTemplateDefs {
		t, err := html.ParseFiles(files...)
		if err != nil {
			return fmt.Errorf("Unable to parse templates for %s: %v", key, err)
		}

		serverTemplates[key] = t
	}

	// Parse client templates
	//for key, def := range chatTemplateDefs {
	//	t := text.New(key)
	//	err, _ := t.Parse(def)
	//	if err != nil {
	//		return fmt.Errorf("Unabel to parse chat template %q: %v", key, err)
	//	}

	//	chatTemplates[key] = t
	//}

	return nil
}

// TODO
func LoadChatTemplates() error {
	return nil
}

func ExecuteChatTemplate(typeA, typeB int, data interface{}) (string, error) {
	key := fmt.Sprint(typeA, typeB)
	t := chatTemplates[key]
	builder := &strings.Builder{}

	if err := t.Execute(builder, data); err != nil {
		return "", err
	}

	return builder.String(), nil
}

func ExecuteServerTemplate(w http.ResponseWriter, key string, data interface{}) error {
	t, ok := serverTemplates[key]
	if !ok {
		return fmt.Errorf("Template with the key %q does not exist", key)
	}

	return t.Execute(w, data)
}
