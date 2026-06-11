package fm

import (
	"encoding/json"
	"testing"
)

func TestDiscoverSchemaRequestJSON(t *testing.T) {
	opts := DefaultDiscoveryOptions()
	req := DiscoverSchemaRequest{
		Documents: []DiscoveryDocument{{
			ID: "doc-1",
			Source: DiscoveryDocumentSource{
				Type:    "text",
				Content: "請求日 2026-01-01",
			},
		}},
		Options: &opts,
	}
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	docs := got["documents"].([]any)
	source := docs[0].(map[string]any)["source"].(map[string]any)
	if source["type"] != "text" {
		t.Fatalf("source type = %v", source["type"])
	}
	options := got["options"].(map[string]any)
	if options["min_presence_rate_for_required"] == nil {
		t.Fatal("missing min_presence_rate_for_required")
	}
}

func TestExportSchemaJSONSchema(t *testing.T) {
	resp, err := ExportSchema(ExportSchemaRequest{
		Format: "json_schema",
		SchemaCandidate: SchemaCandidate{
			ID:      "schema-1",
			Name:    "Invoice",
			Version: "0.1.0",
			Format:  "json_schema",
			Status:  "candidate",
			Schema:  map[string]any{"type": "object"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Format != "json_schema" || resp.Content == "" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}
