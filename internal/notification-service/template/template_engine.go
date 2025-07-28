package template

import (
	"bytes"
	"fmt"
	htmlTemplate "html/template"
	textTemplate "text/template"
)

// TemplateEngine 模板引擎
type TemplateEngine struct {
	textTemplate *textTemplate.Template
	htmlTemplate *htmlTemplate.Template
}

// NewTemplateEngine 创建新的模板引擎
func NewTemplateEngine() *TemplateEngine {
	return &TemplateEngine{
		textTemplate: textTemplate.New("notification"),
		htmlTemplate: htmlTemplate.New("notification"),
	}
}

// RenderText 渲染文本模板
func (te *TemplateEngine) RenderText(templateStr string, data interface{}) (string, error) {
	tmpl, err := te.textTemplate.Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse text template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute text template: %w", err)
	}

	return buf.String(), nil
}

// RenderHTML 渲染HTML模板
func (te *TemplateEngine) RenderHTML(templateStr string, data interface{}) (string, error) {
	tmpl, err := te.htmlTemplate.Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute HTML template: %w", err)
	}

	return buf.String(), nil
}
