package openapi

import "testing"

func TestPropertyTypeParenthesizesEnumArrays(t *testing.T) {
	raw := map[string]any{
		"type": "array",
		"items": map[string]any{
			"type": "string",
			"enum": []any{"GUEST", "USER", "ADMIN"},
		},
	}

	got := propertyType(raw, nil)
	want := "(\"GUEST\" | \"USER\" | \"ADMIN\")[]"
	if got != want {
		t.Fatalf("propertyType() = %q, want %q", got, want)
	}
}
