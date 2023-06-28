package prompt

import (
	"bytes"
	"text/template"
	"text/template/parse"

	"github.com/Masterminds/sprig/v3"
)

type Formatter struct {
	text     string
	template *template.Template
	fields   []string
}

func NewFormatter(text string) *Formatter {
	t := template.Must(template.New("template").Funcs(sprig.FuncMap()).Parse(text))

	return &Formatter{
		text:     text,
		template: t,
		fields:   ListTemplateFields(t),
	}
}

func (pt *Formatter) Render(values map[string]any) (string, error) {
	var doc bytes.Buffer
	if err := pt.template.Execute(&doc, values); err != nil {
		return "", err
	}

	return doc.String(), nil
}

func (pt *Formatter) Fields() []string {
	return pt.fields
}

func ListTemplateFields(t *template.Template) []string {
	return listNodeFields(t.Tree.Root)
}

func listNodeFields(node parse.Node) []string {
	res := []string{}
	if node.Type() == parse.NodeAction {
		res = append(res, node.String())
	}

	if ln, ok := node.(*parse.ListNode); ok {
		for _, n := range ln.Nodes {
			res = append(res, listNodeFields(n)...)
		}
	}

	return res
}
