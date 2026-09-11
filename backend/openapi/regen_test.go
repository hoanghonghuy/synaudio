package openapi_test

import (
	"os"
	"testing"

	"github.com/synaudio/synaudio/backend/openapi"
)

func TestRegenerateAPIContract(t *testing.T) {
	if os.Getenv("REGEN_OPENAPI") == "" {
		t.Skip("set REGEN_OPENAPI=1 to regenerate backend/openapi/api.yaml from bindings and schemas.yaml")
	}
	doc, err := openapi.LoadDocument("api.yaml")
	if err != nil {
		t.Fatalf("load api.yaml: %v", err)
	}
	if err := openapi.EnrichDocument(doc); err != nil {
		t.Fatalf("enrich api.yaml: %v", err)
	}
	if err := openapi.SaveDocument("api.yaml", doc); err != nil {
		t.Fatalf("save api.yaml: %v", err)
	}
}
