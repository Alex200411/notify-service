package template

import (
	"bytes"
	"fmt"
	"text/template"
)

func RenderBody(tmpl string, payload map[string]interface{}) (string, error) {
	if tmpl == "" {
		return "", nil
	}
	t, err := template.New("body").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("invalid body template: %w", err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, payload); err != nil {
		return "", fmt.Errorf("template render failed: %w", err)
	}
	return buf.String(), nil
}
