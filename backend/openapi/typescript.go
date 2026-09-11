package openapi

import (
	"fmt"
	"sort"
	"strings"
)

var httpMethodsSet = map[string]struct{}{
	"get": {}, "post": {}, "put": {}, "delete": {}, "patch": {},
}

// GenerateFrontendContract renders the frontend boundary from an enriched OpenAPI document.
func GenerateFrontendContract(doc *Document) string {
	ops := collectOperations(doc)
	var b strings.Builder
	b.WriteString("// Code generated from backend/openapi/api.yaml contract surface. DO NOT EDIT.\n")
	b.WriteString("// backend/openapi/contract_test.go verifies this file stays synchronized.\n\n")

	b.WriteString("export const API_OPERATIONS = [\n")
	for _, op := range ops {
		fmt.Fprintf(&b, "  { method: %q, path: %q, operationId: %q },\n", op.Method, op.Path, op.ID)
	}
	b.WriteString("] as const\n\n")
	b.WriteString("export type ApiOperation = (typeof API_OPERATIONS)[number]\n")
	b.WriteString("export type ApiMethod = ApiOperation[\"method\"]\n")
	b.WriteString("export type ApiPath = ApiOperation[\"path\"]\n")
	b.WriteString("export type ApiOperationId = ApiOperation[\"operationId\"]\n\n")

	schemas := extractSchemas(doc)
	names := sortedKeys(schemas)
	for _, name := range names {
		writeTSType(&b, name, schemas[name], schemas)
		b.WriteString("\n")
	}

	return b.String()
}

type apiOperation struct {
	Method string
	Path   string
	ID     string
}

func collectOperations(doc *Document) []apiOperation {
	ops := make([]apiOperation, 0)
	for path, item := range doc.Paths {
		for method, raw := range item {
			if _, ok := httpMethodsSet[strings.ToLower(method)]; !ok {
				continue
			}
			op, _ := raw.(map[string]any)
			id, _ := op["operationId"].(string)
			ops = append(ops, apiOperation{Method: strings.ToUpper(method), Path: path, ID: id})
		}
	}
	sort.Slice(ops, func(i, j int) bool {
		if ops[i].Path == ops[j].Path {
			return ops[i].Method < ops[j].Method
		}
		return ops[i].Path < ops[j].Path
	})
	return ops
}

func extractSchemas(doc *Document) map[string]any {
	if doc.Components == nil {
		return map[string]any{}
	}
	schemas, _ := doc.Components["schemas"].(map[string]any)
	if schemas == nil {
		return map[string]any{}
	}
	return schemas
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func writeTSType(b *strings.Builder, name string, schema any, all map[string]any) {
	obj, ok := schema.(map[string]any)
	if !ok {
		return
	}
	if enum := stringEnum(obj); enum != "" {
		fmt.Fprintf(b, "export type %s = %s\n", name, enum)
		return
	}
	if obj["type"] != "object" && obj["properties"] == nil {
		if ts := scalarType(obj); ts != "" {
			fmt.Fprintf(b, "export type %s = %s\n", name, ts)
			return
		}
	}
	fmt.Fprintf(b, "export interface %s {\n", name)
	props, _ := obj["properties"].(map[string]any)
	required := requiredSet(obj["required"])
	if props != nil {
		propNames := sortedKeys(props)
		for _, prop := range propNames {
			optional := ""
			if !required[prop] {
				optional = "?"
			}
			fmt.Fprintf(b, "  %s%s: %s\n", prop, optional, propertyType(props[prop], all))
		}
	}
	b.WriteString("}\n")
}

func requiredSet(raw any) map[string]bool {
	set := map[string]bool{}
	items, ok := raw.([]any)
	if !ok {
		return set
	}
	for _, item := range items {
		if s, ok := item.(string); ok {
			set[s] = true
		}
	}
	return set
}

func stringEnum(obj map[string]any) string {
	if obj["type"] != "string" {
		return ""
	}
	raw, ok := obj["enum"].([]any)
	if !ok || len(raw) == 0 {
		return ""
	}
	parts := make([]string, 0, len(raw))
	for _, v := range raw {
		parts = append(parts, fmt.Sprintf("%q", v))
	}
	return strings.Join(parts, " | ")
}

func scalarType(obj map[string]any) string {
	switch obj["type"] {
	case "string":
		return "string"
	case "integer", "number":
		return "number"
	case "boolean":
		return "boolean"
	default:
		return ""
	}
}

func propertyType(raw any, all map[string]any) string {
	obj, ok := raw.(map[string]any)
	if !ok {
		return "unknown"
	}
	if ref, ok := obj["$ref"].(string); ok {
		return refTypeName(ref)
	}
	if enum := stringEnum(obj); enum != "" {
		return enum
	}
	switch obj["type"] {
	case "string":
		return "string"
	case "integer", "number":
		return "number"
	case "boolean":
		return "boolean"
	case "array":
		items, _ := obj["items"].(map[string]any)
		if items == nil {
			return "unknown[]"
		}
		itemType := propertyType(items, all)
		if strings.Contains(itemType, " | ") {
			itemType = "(" + itemType + ")"
		}
		return itemType + "[]"
	case "object":
		if obj["additionalProperties"] != nil {
			return "Record<string, unknown>"
		}
		return "Record<string, unknown>"
	default:
		return "unknown"
	}
}

func refTypeName(ref string) string {
	const prefix = "#/components/schemas/"
	if strings.HasPrefix(ref, prefix) {
		return ref[len(prefix):]
	}
	return "unknown"
}
