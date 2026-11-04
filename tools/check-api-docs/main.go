// Command check-api-docs compares registered HTTP routes with Swagger and Postman.
package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type route struct{ method, path string }
type operation struct {
	Security  []map[string][]string `json:"security"`
	Responses map[string]any        `json:"responses"`
}
type collectionItem struct {
	Item    []collectionItem `json:"item"`
	Request *struct {
		Method string          `json:"method"`
		URL    json.RawMessage `json:"url"`
	} `json:"request"`
}

var parameter = regexp.MustCompile(`:[A-Za-z_][A-Za-z0-9_]*|\{[^{}]+\}`)

func normalize(path string) string {
	path = strings.TrimPrefix(path, "{{host}}")
	path, _, _ = strings.Cut(path, "?")
	return parameter.ReplaceAllString(path, "{param}")
}

func literal(expr ast.Expr) string {
	v, ok := expr.(*ast.BasicLit)
	if !ok || v.Kind != token.STRING {
		return ""
	}
	value, _ := strconv.Unquote(v.Value)
	return value
}

func collectRoutes(filename string, source any) (map[route]string, error) {
	file, err := parser.ParseFile(token.NewFileSet(), filename, source, 0)
	if err != nil {
		return nil, err
	}
	routes := map[route]string{}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		groups := map[string]string{"eng": "", "r": ""}
		auth := map[string]string{}
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			if assign, ok := node.(*ast.AssignStmt); ok {
				for i, expr := range assign.Rhs {
					call, ok := expr.(*ast.CallExpr)
					if !ok || i >= len(assign.Lhs) || len(call.Args) == 0 {
						continue
					}
					sel, ok := call.Fun.(*ast.SelectorExpr)
					if !ok || sel.Sel.Name != "Group" {
						continue
					}
					parent, pok := sel.X.(*ast.Ident)
					name, nok := assign.Lhs[i].(*ast.Ident)
					if pok && nok {
						groups[name.Name] = groups[parent.Name] + literal(call.Args[0])
						auth[name.Name] = auth[parent.Name]
					}
				}
			}
			call, ok := node.(*ast.CallExpr)
			if !ok || len(call.Args) == 0 {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			receiver, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}
			base, known := groups[receiver.Name]
			if !known {
				return true
			}
			if sel.Sel.Name == "Use" {
				for _, arg := range call.Args {
					if middleware, ok := arg.(*ast.SelectorExpr); ok {
						switch middleware.Sel.Name {
						case "AuthAdmin":
							auth[receiver.Name] = "AdminApiKey"
						case "Auth":
							auth[receiver.Name] = "InstanceApiKey"
						}
					}
				}
			}
			switch sel.Sel.Name {
			case "GET", "POST", "PUT", "PATCH", "DELETE":
				path := base + literal(call.Args[0])
				// Browser assets and the WebSocket upgrade are documented in guides.
				if path == "/ws" || path == "/favicon.ico" || strings.HasPrefix(path, "/manager") || strings.HasPrefix(path, "/swagger") {
					return true
				}
				routes[route{strings.ToLower(sel.Sel.Name), normalize(path)}] = auth[receiver.Name]
			}
			return true
		})
	}
	return routes, nil
}

func readJSON(filename string, target any) {
	data, err := os.ReadFile(filename)
	if err == nil {
		err = json.Unmarshal(data, target)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func main() {
	actual := map[route]string{}
	for _, filename := range []string{"pkg/routes/routes.go", "cmd/evolution-go/main.go"} {
		routes, err := collectRoutes(filename, nil)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		for route, security := range routes {
			actual[route] = security
		}
	}
	var swagger struct {
		Info struct {
			Version string `json:"version"`
		} `json:"info"`
		Paths       map[string]map[string]operation `json:"paths"`
		Definitions map[string]struct {
			Properties map[string]struct {
				Type string `json:"type"`
			} `json:"properties"`
		} `json:"definitions"`
	}
	readJSON("docs/swagger.json", &swagger)
	documented := map[route]operation{}
	for path, methods := range swagger.Paths {
		for method, op := range methods {
			documented[route{method, normalize(path)}] = op
		}
	}
	var collection struct {
		Item []collectionItem `json:"item"`
	}
	readJSON("Evolution GO.postman_collection.json", &collection)
	postman := map[route]bool{}
	var walk func([]collectionItem)
	walk = func(items []collectionItem) {
		for _, item := range items {
			walk(item.Item)
			if item.Request == nil {
				continue
			}
			var raw string
			if json.Unmarshal(item.Request.URL, &raw) != nil {
				var url struct{ Raw string }
				if err := json.Unmarshal(item.Request.URL, &url); err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}
				raw = url.Raw
			}
			postman[route{strings.ToLower(item.Request.Method), normalize(raw)}] = true
		}
	}
	walk(collection.Item)
	var failures []string
	// Go fields without JSON tags retain their exact names on the wire.
	// JID implements TextMarshaler, so its internal Go fields are not the API shape.
	for name, schema := range swagger.Definitions {
		if strings.HasSuffix(name, ".MessageSendStruct") {
			for _, field := range []string{"Info", "Message", "MessageContextInfo"} {
				if _, ok := schema.Properties[field]; !ok {
					failures = append(failures, "Swagger must preserve JSON field "+field+"; generate with --propertyStrategy pascalcase")
				}
			}
		}
		if strings.HasSuffix(name, ".MessageInfo") && schema.Properties["Chat"].Type != "string" {
			failures = append(failures, "Swagger MessageInfo.Chat must be a string; check .swaggo and --propertyStrategy pascalcase")
		}
	}
	version, err := os.ReadFile("VERSION")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if swagger.Info.Version != strings.TrimSpace(string(version)) {
		failures = append(failures, "Swagger info.version differs from VERSION; update @version in main.go and regenerate")
	}
	for key, security := range actual {
		op, ok := documented[key]
		if !ok {
			failures = append(failures, fmt.Sprintf("Swagger missing: %s %s", key.method, key.path))
		} else if security != "" {
			found := false
			for _, alternative := range op.Security {
				_, exists := alternative[security]
				found = found || exists
			}
			if !found {
				failures = append(failures, fmt.Sprintf("Wrong Swagger authentication: %s %s (want %s)", key.method, key.path, security))
			}
			for _, status := range []string{"401"} {
				if _, exists := op.Responses[status]; !exists {
					failures = append(failures, fmt.Sprintf("Swagger missing HTTP %s: %s %s", status, key.method, key.path))
				}
			}
		} else if len(op.Security) != 0 {
			failures = append(failures, fmt.Sprintf("Public endpoint marked as authenticated: %s %s", key.method, key.path))
		}
		if !postman[key] {
			failures = append(failures, fmt.Sprintf("Postman missing: %s %s", key.method, key.path))
		}
	}
	for key := range documented {
		if _, ok := actual[key]; !ok {
			failures = append(failures, fmt.Sprintf("Swagger documents an unregistered route: %s %s", key.method, key.path))
		}
	}
	for key := range postman {
		if _, ok := actual[key]; !ok {
			failures = append(failures, fmt.Sprintf("Postman documents an unregistered route: %s %s", key.method, key.path))
		}
	}
	if len(failures) != 0 {
		sort.Strings(failures)
		fmt.Fprintln(os.Stderr, strings.Join(failures, "\n"))
		os.Exit(1)
	}
	fmt.Printf("API documentation matches %d HTTP operations; authentication, 401 and Postman coverage checked.\n", len(actual))
}
