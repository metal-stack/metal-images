package templates

import (
	"bytes"
	_ "embed"
	"text/template"

	"github.com/metal-stack/metal-go/api/models"
)

type Chrony struct {
	NTPServers []*models.MetalNTPServer
}

//go:embed chrony.conf.tpl
var chronyTemplate string

func RenderChronyTemplate(chronyConfig Chrony) (string, error) {
	templ, err := template.New("chrony").Parse(chronyTemplate)
	if err != nil {
		return "error parsing template", err
	}

	rendered := new(bytes.Buffer)
	err = templ.Execute(rendered, chronyConfig)
	if err != nil {
		return "error writing to template file", err
	}
	return rendered.String(), nil
}
