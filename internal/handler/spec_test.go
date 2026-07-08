package handler

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/sprungknoedl/dagobert/internal/model"
	"gopkg.in/yaml.v3"
)

// TestOpenAPISpecDrift keeps openapi.yaml honest against the actual route
// table and JSON-capable handlers without hand-maintaining a route list or
// relying on generated/annotated code. It parses this package's source with
// go/parser rather than spinning up the mux, so it needs no server, database
// or fixtures.
func TestOpenAPISpecDrift(t *testing.T) {
	registrations := parseRouteRegistrations(t, "init.go")
	jsonCapable := parseJSONCapableHandlers(t)
	spec := loadOpenAPISpec(t, "../../openapi.yaml")

	specOps := map[string]bool{}
	for path, methods := range spec.paths() {
		for method := range methods {
			switch method {
			case "get", "post", "delete", "put", "patch":
				specOps[strings.ToUpper(method)+" "+path] = true
			}
		}
	}

	// forward: every openapi.yaml operation must be a registered route.
	forward := make([]string, 0, len(specOps))
	for op := range specOps {
		forward = append(forward, op)
	}
	sort.Strings(forward)
	for _, op := range forward {
		if _, ok := registrations[op]; !ok {
			t.Errorf("forward: openapi.yaml operation %q has no matching route registration in init.go", op)
		}
	}

	// reverse: every registered route whose handler is JSON-capable must be documented.
	reverse := make([]string, 0, len(registrations))
	for op := range registrations {
		reverse = append(reverse, op)
	}
	sort.Strings(reverse)
	for _, op := range reverse {
		handler := registrations[op]
		if jsonCapable[handler] && !specOps[op] {
			t.Errorf("reverse: %s (handler %s) answers JSON but is missing from openapi.yaml", op, handler)
		}
	}

	// schema: component properties must match the model struct fields they document.
	schemaModels := map[string]any{
		"Case":      model.Case{},
		"Asset":     model.Asset{},
		"Event":     model.Event{},
		"Evidence":  model.Evidence{},
		"Indicator": model.Indicator{},
		"Malware":   model.Malware{},
		"Note":      model.Note{},
		"Task":      model.Task{},
		"Comment":   model.Comment{},
	}
	names := make([]string, 0, len(schemaModels))
	for name := range schemaModels {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		props, ok := spec.schemaProperties(name)
		if !ok {
			t.Errorf("schema: openapi.yaml has no components.schemas.%s", name)
			continue
		}

		goFields := map[string]bool{}
		rt := reflect.TypeOf(schemaModels[name])
		for i := 0; i < rt.NumField(); i++ {
			f := rt.Field(i)
			if f.Tag.Get("json") == "-" {
				continue
			}
			goFields[f.Name] = true
		}

		for f := range goFields {
			if !props[f] {
				t.Errorf("schema %s: model field %s is missing from openapi.yaml", name, f)
			}
		}
		for f := range props {
			if !goFields[f] {
				t.Errorf("schema %s: openapi.yaml has property %s with no matching model field", name, f)
			}
		}
	}
}

// parseRouteRegistrations extracts "METHOD /path" -> handler-method-name from
// `mux.HandleFunc("METHOD /path", h.Handler)` call sites in path.
func parseRouteRegistrations(t *testing.T, path string) map[string]string {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	routes := map[string]string{}
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "HandleFunc" || len(call.Args) != 2 {
			return true
		}
		lit, ok := call.Args[0].(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		pattern, err := strconv.Unquote(lit.Value)
		if err != nil {
			return true
		}
		handlerSel, ok := call.Args[1].(*ast.SelectorExpr)
		if !ok {
			return true
		}
		routes[pattern] = handlerSel.Sel.Name
		return true
	})
	return routes
}

// parseJSONCapableHandlers scans every .go file in this package (excluding
// tests) for function bodies that answer JSON: a Render call with a non-nil
// data argument, a RedirectAfterSave call with a non-nil record argument, or a
// 204 (the marker every JSON-aware delete handler writes on the confirmed
// path).
func parseJSONCapableHandlers(t *testing.T) map[string]bool {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package dir: %v", err)
	}

	capable := map[string]bool{}
	fset := token.NewFileSet()
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			if isJSONCapable(fn.Body) {
				capable[fn.Name.Name] = true
			}
		}
	}
	return capable
}

func isJSONCapable(body *ast.BlockStmt) bool {
	capable := false
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch fun := call.Fun.(type) {
		case *ast.Ident:
			switch fun.Name {
			case "Render":
				if len(call.Args) == 5 && !isNilIdent(call.Args[4]) {
					capable = true
				}
			case "RedirectAfterSave":
				if len(call.Args) == 4 && !isNilIdent(call.Args[3]) {
					capable = true
				}
			}
		case *ast.SelectorExpr:
			if fun.Sel.Name == "WriteHeader" && len(call.Args) == 1 {
				if arg, ok := call.Args[0].(*ast.SelectorExpr); ok && arg.Sel.Name == "StatusNoContent" {
					capable = true
				}
			}
		}
		return true
	})
	return capable
}

func isNilIdent(e ast.Expr) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == "nil"
}

// openAPISpec is the minimal shape of openapi.yaml this test needs. raw is
// deliberately an unnamed map[string]any (not a defined type): yaml.v3
// remembers the concrete type of the outermost map[string]any it decodes into
// and reuses it for every nested mapping, so a defined type here would make
// every nested map come back as that defined type instead of plain
// map[string]any, breaking the type assertions below.
type openAPISpec struct {
	raw map[string]any
}

func loadOpenAPISpec(t *testing.T, path string) openAPISpec {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return openAPISpec{raw: raw}
}

func (s openAPISpec) paths() map[string]map[string]any {
	out := map[string]map[string]any{}
	raw, _ := s.raw["paths"].(map[string]any)
	for path, methods := range raw {
		m, ok := methods.(map[string]any)
		if !ok {
			continue
		}
		out[path] = m
	}
	return out
}

func (s openAPISpec) schemaProperties(name string) (map[string]bool, bool) {
	components, _ := s.raw["components"].(map[string]any)
	schemas, _ := components["schemas"].(map[string]any)
	schema, ok := schemas[name].(map[string]any)
	if !ok {
		return nil, false
	}
	props, _ := schema["properties"].(map[string]any)
	out := map[string]bool{}
	for k := range props {
		out[k] = true
	}
	return out, true
}
