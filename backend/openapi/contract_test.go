package openapi_test

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/synaudio/synaudio/backend/openapi"
	"go.yaml.in/yaml/v3"
)

type routeSource struct {
	path   string
	prefix string
}

type openAPIDocument struct {
	Paths      map[string]map[string]any `yaml:"paths"`
	Components struct {
		SecuritySchemes map[string]any `yaml:"securitySchemes"`
		Schemas         map[string]any `yaml:"schemas"`
	} `yaml:"components"`
}

var routeRegistration = regexp.MustCompile(`\br\.(Get|Post|Put|Delete|Patch)\("([^"]+)"`)
var pathParameter = regexp.MustCompile(`\{([^}]+)\}`)

var httpMethods = map[string]struct{}{
	"get": {}, "post": {}, "put": {}, "delete": {}, "patch": {},
}

func TestRegenerateFrontendContract(t *testing.T) {
	if os.Getenv("REGEN_OPENAPI_FRONTEND") == "" {
		t.Skip("set REGEN_OPENAPI_FRONTEND=1 to regenerate frontend/src/api/openapi.generated.ts")
	}
	doc, err := openapi.LoadDocument("api.yaml")
	if err != nil {
		t.Fatalf("load OpenAPI contract: %v", err)
	}
	wantTS := openapi.GenerateFrontendContract(doc)
	if err := os.WriteFile("../../frontend/src/api/openapi.generated.ts", []byte(wantTS), 0o644); err != nil {
		t.Fatalf("write generated frontend contract: %v", err)
	}
}

func TestOpenAPIContract(t *testing.T) {
	doc := loadDocument(t)
	if err := openapi.ValidateOperationSchemas(toPackageDocument(doc)); err != nil {
		t.Fatal(err)
	}
	runtime := runtimeRoutes(t)
	documented := documentedRoutes(doc)

	if missing := difference(runtime, documented); len(missing) > 0 {
		t.Fatalf("OpenAPI missing runtime operations: %v", missing)
	}
	if stale := difference(documented, runtime); len(stale) > 0 {
		t.Fatalf("OpenAPI contains stale operations: %v", stale)
	}

	if _, ok := doc.Components.SecuritySchemes["bearerAuth"]; !ok {
		t.Fatal("components.securitySchemes.bearerAuth is required")
	}
	if _, ok := doc.Components.Schemas["ErrorResponse"]; !ok {
		t.Fatal("components.schemas.ErrorResponse is required")
	}
	validateBindingCoverage(t, doc)

	operationIDs := map[string]string{}
	for path, item := range doc.Paths {
		validatePathParameters(t, path, item)
		for method, raw := range item {
			if _, ok := httpMethods[strings.ToLower(method)]; !ok {
				continue
			}
			op, ok := raw.(map[string]any)
			if !ok {
				t.Fatalf("%s %s operation must be an object", strings.ToUpper(method), path)
			}
			id, _ := op["operationId"].(string)
			if strings.TrimSpace(id) == "" {
				t.Fatalf("%s %s missing operationId", strings.ToUpper(method), path)
			}
			if previous, exists := operationIDs[id]; exists {
				t.Fatalf("duplicate operationId %q on %s and %s %s", id, previous, strings.ToUpper(method), path)
			}
			operationIDs[id] = strings.ToUpper(method) + " " + path
			validateSecurity(t, path, strings.ToUpper(method), op)
		}
	}

	pkgDoc, err := openapi.LoadDocument("api.yaml")
	if err != nil {
		t.Fatalf("reload OpenAPI contract: %v", err)
	}
	wantTS := openapi.GenerateFrontendContract(pkgDoc)
	gotTS, err := os.ReadFile("../../frontend/src/api/openapi.generated.ts")
	if err != nil {
		t.Fatalf("read generated frontend contract: %v", err)
	}
	if string(gotTS) != wantTS {
		t.Fatal("frontend/src/api/openapi.generated.ts drifted from backend/openapi/api.yaml")
	}
}

func toPackageDocument(doc openAPIDocument) *openapi.Document {
	return &openapi.Document{
		Paths: doc.Paths,
		Components: map[string]any{
			"schemas":         doc.Components.Schemas,
			"securitySchemes": doc.Components.SecuritySchemes,
			"responses":       map[string]any{},
		},
	}
}

func validateBindingCoverage(t *testing.T, doc openAPIDocument) {
	t.Helper()
	seen := map[string]struct{}{}
	for _, item := range doc.Paths {
		for method, raw := range item {
			if _, ok := httpMethods[strings.ToLower(method)]; !ok {
				continue
			}
			op, _ := raw.(map[string]any)
			id, _ := op["operationId"].(string)
			if id == "" {
				continue
			}
			seen[id] = struct{}{}
			if _, ok := openapi.OperationBindings[id]; !ok {
				t.Fatalf("missing OperationBinding for operationId %q", id)
			}
		}
	}
	if len(seen) != len(openapi.OperationBindings) {
		t.Fatalf("OperationBindings has %d entries but api.yaml declares %d operationIds", len(openapi.OperationBindings), len(seen))
	}
}

func loadDocument(t *testing.T) openAPIDocument {
	t.Helper()
	body, err := os.ReadFile("api.yaml")
	if err != nil {
		t.Fatalf("read OpenAPI contract: %v", err)
	}
	var doc openAPIDocument
	if err := yaml.Unmarshal(body, &doc); err != nil {
		t.Fatalf("parse OpenAPI YAML: %v", err)
	}
	if len(doc.Paths) == 0 {
		t.Fatal("OpenAPI paths must not be empty")
	}
	return doc
}

func runtimeRoutes(t *testing.T) map[string]struct{} {
	t.Helper()
	sources := []routeSource{
		{path: "../internal/platform/httpapi/router.go"},
		{path: "../internal/identity/handler.go", prefix: "/api/v1/auth"},
		{path: "../internal/identity/admin_handler.go", prefix: "/api/v1"},
		{path: "../internal/story/handler.go", prefix: "/api/v1"},
		{path: "../internal/story/readiness_handler.go", prefix: "/api/v1"},
		{path: "../internal/planning/handler.go", prefix: "/api/v1"},
		{path: "../internal/planning/workspace_handler.go", prefix: "/api/v1"},
		{path: "../internal/generation/handler.go", prefix: "/api/v1"},
		{path: "../internal/generation/latest_run_handler.go", prefix: "/api/v1"},
		{path: "../internal/audio/handler.go", prefix: "/api/v1"},
		{path: "../internal/listener/handler.go", prefix: "/api/v1"},
		{path: "../internal/retcon/handler.go", prefix: "/api/v1"},
		{path: "../internal/audit/handler.go", prefix: "/api/v1"},
	}
	routes := map[string]struct{}{}
	for _, source := range sources {
		body, err := os.ReadFile(source.path)
		if err != nil {
			t.Fatalf("read route source %s: %v", source.path, err)
		}
		for _, match := range routeRegistration.FindAllStringSubmatch(string(body), -1) {
			routes[strings.ToUpper(match[1])+" "+source.prefix+match[2]] = struct{}{}
		}
	}
	return routes
}

func documentedRoutes(doc openAPIDocument) map[string]struct{} {
	routes := map[string]struct{}{}
	for path, item := range doc.Paths {
		for method := range item {
			if _, ok := httpMethods[strings.ToLower(method)]; ok {
				routes[strings.ToUpper(method)+" "+path] = struct{}{}
			}
		}
	}
	return routes
}

func validatePathParameters(t *testing.T, path string, item map[string]any) {
	t.Helper()
	required := pathParameter.FindAllStringSubmatch(path, -1)
	if len(required) == 0 {
		return
	}
	declared := map[string]bool{}
	params, _ := item["parameters"].([]any)
	for _, raw := range params {
		param, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		name, _ := param["name"].(string)
		in, _ := param["in"].(string)
		isRequired, _ := param["required"].(bool)
		if in == "path" && isRequired {
			declared[name] = true
		}
	}
	for _, match := range required {
		if !declared[match[1]] {
			t.Fatalf("path %s missing required path parameter declaration for %s", path, match[1])
		}
	}
}

func validateSecurity(t *testing.T, path, method string, operation map[string]any) {
	t.Helper()
	if isPublicOperation(path) {
		return
	}
	security, ok := operation["security"].([]any)
	if !ok || len(security) == 0 {
		t.Fatalf("%s %s must declare bearerAuth security", method, path)
	}
	found := false
	for _, raw := range security {
		req, ok := raw.(map[string]any)
		if ok {
			if _, exists := req["bearerAuth"]; exists {
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("%s %s must declare bearerAuth security", method, path)
	}
}

func isPublicOperation(path string) bool {
	switch path {
	case "/health", "/ready",
		"/api/v1/auth/register", "/api/v1/auth/login", "/api/v1/auth/refresh",
		"/api/v1/auth/email/verify", "/api/v1/auth/email/resend",
		"/api/v1/auth/password/forgot", "/api/v1/auth/password/reset",
		"/api/v1/stories", "/api/v1/genres",
		"/api/v1/stories/{storyID}", "/api/v1/stories/{storyID}/chapters",
		"/api/v1/chapters/{chapterID}/content", "/api/v1/chapters/{chapterID}/audio-url":
		return true
	default:
		return false
	}
}

func difference(left, right map[string]struct{}) []string {
	items := make([]string, 0)
	for key := range left {
		if _, ok := right[key]; !ok {
			items = append(items, key)
		}
	}
	sort.Strings(items)
	return items
}
