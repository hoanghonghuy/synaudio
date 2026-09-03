package openapi_test

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
)

type routeSource struct {
	path   string
	prefix string
}

var routeRegistration = regexp.MustCompile(`\br\.(Get|Post|Put|Delete|Patch)\("([^"]+)"`)
var openAPIPath = regexp.MustCompile(`^  (/[^:]+):\s*$`)
var openAPIMethod = regexp.MustCompile(`^    (get|post|put|delete|patch):\s*$`)

// TestOpenAPIMatchesHTTPRouteSurface makes the contract executable: every
// route registered by a production handler must be present in OpenAPI and every
// OpenAPI operation must still exist in the runtime route surface. A public API
// change therefore cannot silently drift from backend/openapi/api.yaml.
func TestOpenAPIMatchesHTTPRouteSurface(t *testing.T) {
	sources := []routeSource{
		{path: "../internal/platform/httpapi/router.go"},
		{path: "../internal/identity/handler.go", prefix: "/api/v1/auth"},
		{path: "../internal/story/handler.go", prefix: "/api/v1"},
		{path: "../internal/planning/handler.go", prefix: "/api/v1"},
		{path: "../internal/generation/handler.go", prefix: "/api/v1"},
		{path: "../internal/audio/handler.go", prefix: "/api/v1"},
		{path: "../internal/listener/handler.go", prefix: "/api/v1"},
		{path: "../internal/retcon/handler.go", prefix: "/api/v1"},
		{path: "../internal/audit/handler.go", prefix: "/api/v1"},
	}

	runtime := map[string]struct{}{}
	for _, source := range sources {
		body, err := os.ReadFile(source.path)
		if err != nil {
			t.Fatalf("read route source %s: %v", source.path, err)
		}
		for _, match := range routeRegistration.FindAllStringSubmatch(string(body), -1) {
			method := strings.ToUpper(match[1])
			path := source.prefix + match[2]
			runtime[method+" "+path] = struct{}{}
		}
	}

	documented, err := readOpenAPIRoutes("api.yaml")
	if err != nil {
		t.Fatal(err)
	}

	missing := difference(runtime, documented)
	stale := difference(documented, runtime)
	if len(missing) > 0 || len(stale) > 0 {
		t.Fatalf("OpenAPI route drift\nmissing operations: %v\nstale operations: %v", missing, stale)
	}
}

func readOpenAPIRoutes(path string) (map[string]struct{}, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open OpenAPI contract: %w", err)
	}
	defer file.Close()

	routes := map[string]struct{}{}
	currentPath := ""
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if match := openAPIPath.FindStringSubmatch(line); match != nil {
			currentPath = match[1]
			continue
		}
		if currentPath == "" {
			continue
		}
		if match := openAPIMethod.FindStringSubmatch(line); match != nil {
			routes[strings.ToUpper(match[1])+" "+currentPath] = struct{}{}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan OpenAPI contract: %w", err)
	}
	return routes, nil
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
