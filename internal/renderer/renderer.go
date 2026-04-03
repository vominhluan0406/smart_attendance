package renderer

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var funcMap = template.FuncMap{
	"containsMethod": func(methods, method string) bool {
		for _, m := range strings.Split(methods, ",") {
			if strings.TrimSpace(m) == method {
				return true
			}
		}
		return false
	},
	"add": func(a, b int) int {
		return a + b
	},
	"subtract": func(a, b int) int {
		return a - b
	},
	"slice": func(s string, start, end int) string {
		if start < 0 {
			start = 0
		}
		if start > len(s) {
			return ""
		}
		if end > len(s) {
			end = len(s)
		}
		if start > end {
			return ""
		}
		return s[start:end]
	},
	"now": func() time.Time {
		return time.Now()
	},
}

type Renderer struct {
	templates map[string]*template.Template
	mu        sync.RWMutex
	baseDir   string
	devMode   bool
}

func New(baseDir string, devMode bool) (*Renderer, error) {
	r := &Renderer{
		templates: make(map[string]*template.Template),
		baseDir:   baseDir,
		devMode:   devMode,
	}

	if err := r.loadTemplates(); err != nil {
		return nil, err
	}

	absPath, _ := filepath.Abs(baseDir)
	log.Printf("[renderer] loaded %d templates from %s (devMode=%v)", len(r.templates), absPath, devMode)
	return r, nil
}

func (r *Renderer) loadTemplates() error {
	// Build into a new map — only swap on success (atomic)
	newTemplates := make(map[string]*template.Template)

	// Resolve absolute path for clear error messages
	absBase, _ := filepath.Abs(r.baseDir)

	layouts, err := filepath.Glob(filepath.Join(r.baseDir, "layouts", "*.html"))
	if err != nil {
		return fmt.Errorf("glob layouts: %w", err)
	}

	components, err := filepath.Glob(filepath.Join(r.baseDir, "components", "*.html"))
	if err != nil {
		return fmt.Errorf("glob components: %w", err)
	}

	pages, err := filepath.Glob(filepath.Join(r.baseDir, "pages", "*.html"))
	if err != nil {
		return fmt.Errorf("glob pages: %w", err)
	}

	partials, err := filepath.Glob(filepath.Join(r.baseDir, "partials", "*.html"))
	if err != nil {
		return fmt.Errorf("glob partials: %w", err)
	}

	if len(pages) == 0 {
		return fmt.Errorf("no page templates found in %s/pages/ — check working directory (cwd should be project root)", absBase)
	}
	if len(layouts) == 0 {
		return fmt.Errorf("no layout templates found in %s/layouts/", absBase)
	}

	// Shared = layouts + components + partials
	var shared []string
	shared = append(shared, layouts...)
	shared = append(shared, components...)
	shared = append(shared, partials...)

	// Parse each page template: base.html is the entry point, page overrides {{define "body"}}
	for _, page := range pages {
		name := filepath.Base(page)
		files := make([]string, 0, len(shared)+1)
		files = append(files, shared...)
		files = append(files, page)

		tmpl, err := template.New(filepath.Base(files[0])).Funcs(funcMap).ParseFiles(files...)
		if err != nil {
			log.Printf("[renderer] ERROR parsing page %s: %v", name, err)
			return fmt.Errorf("parse page %s: %w", name, err)
		}

		newTemplates[name] = tmpl
	}

	// Parse each partial as standalone (for HTMX fragment responses)
	for _, partial := range partials {
		name := "partials/" + filepath.Base(partial)

		files := make([]string, 0, len(components)+1)
		files = append(files, partial)
		files = append(files, components...)

		tmpl, err := template.New(filepath.Base(partial)).Funcs(funcMap).ParseFiles(files...)
		if err != nil {
			log.Printf("[renderer] ERROR parsing partial %s: %v", name, err)
			return fmt.Errorf("parse partial %s: %w", name, err)
		}

		newTemplates[name] = tmpl
	}

	// Atomic swap — old templates remain if parsing fails
	r.templates = newTemplates
	return nil
}

// Render renders a full page template (from pages/)
func (r *Renderer) Render(w io.Writer, name string, data interface{}) error {
	if r.devMode {
		r.mu.Lock()
		if err := r.loadTemplates(); err != nil {
			r.mu.Unlock()
			log.Printf("[renderer] devMode reload failed: %v", err)
			// Fall through to use previously loaded templates
		} else {
			r.mu.Unlock()
		}
	}

	r.mu.RLock()
	tmpl, ok := r.templates[name]
	r.mu.RUnlock()

	if !ok {
		return fmt.Errorf("template %q not found", name)
	}

	return tmpl.Execute(w, data)
}

// RenderPartial renders an HTMX partial fragment (from partials/).
func (r *Renderer) RenderPartial(w io.Writer, name string, data interface{}) error {
	return r.Render(w, "partials/"+name, data)
}
