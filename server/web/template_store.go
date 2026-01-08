package web

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/rs/zerolog/log"
)

type TemplateStore struct {
	templates map[string]*template.Template
}

func NewTemplateStore() (*TemplateStore, error) {
	tpls := make(map[string]*template.Template)
	pages := []string{"login", "signup", "profile", "profile_edit"}
	for _, p := range pages {
		tpl, err := template.ParseFiles(
			"server/templates/layout.html",
			filepath.Join("server/templates", p+".html"),
		)
		if err != nil {
			return nil, fmt.Errorf("parse templates for %s: %w", p, err)
		}
		tpls[p+".html"] = tpl
	}
	return &TemplateStore{templates: tpls}, nil
}

func (ts *TemplateStore) Render(w http.ResponseWriter, name string, data any) {
	tpl, ok := ts.templates[name]
	if !ok {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}
	if err := tpl.ExecuteTemplate(w, name, data); err != nil {
		log.Error().Err(err).Msg("template render failed")
		http.Error(w, "server error", http.StatusInternalServerError)
	}
}
