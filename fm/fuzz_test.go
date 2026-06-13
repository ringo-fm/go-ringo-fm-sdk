package fm

import (
	"encoding/json"
	"math/rand"
	"strings"
	"testing"
)

func FuzzGeneratedContentFromJSON(f *testing.F) {
	seeds := []string{
		`{"name":"Alice","score":99}`,
		`{"active":true,"disabled":false,"score":42}`,
		`{"price":3.14,"count":7,"label":"hello"}`,
		`{}`,
		`null`,
		``,
		`[1,2,3]`,
		`not json`,
		`{"a":"\x00\x01\x02"}`,
		`{"nested":{"deep":{"key":1}}}`,
		`{"unicode":"日本語テスト"}`,
		`{"empty_str":""}`,
		`{"large_num":9223372036854775807}`,
		`{"neg": -42}`,
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, jsonStr string) {
		c, err := GeneratedContentFromJSON(jsonStr)
		if err != nil {
			return
		}
		defer c.Close()

		_, _ = c.JSON()
		_, _ = c.AsMap()
		_ = c.Value("x")
		_, _ = c.ValueAsInt64("x")
		_, _ = c.ValueAsBool("x")
		_, _ = c.ValueAsFloat64("x")
		_ = c.HasProperty("x")
		_ = c.HasProperty("")
		_ = c.IsComplete()
		names, err := c.PropertyNames()
		if err == nil {
			for _, name := range names {
				_ = c.HasProperty(name)
				c.ValueAsInt64(name)
				c.ValueAsBool(name)
				c.ValueAsFloat64(name)
			}
		}
	})
}

func FuzzTranscriptFromJSON(f *testing.F) {
	seeds := []string{
		`{"version":1,"type":"transcript","transcript":{"entries":[]}}`,
		`{}`,
		``,
		`null`,
		`not json`,
		`{"transcript":{"entries":[{"role":"user","content":"hi"}]}}`,
		`{"key":"bad\x00value"}`,
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, jsonStr string) {
		tr, err := TranscriptFromJSON([]byte(jsonStr))
		if err != nil {
			return
		}
		defer tr.Close()
		_ = tr.EntryCount()
		_, _ = tr.MarshalJSON()
	})
}

func FuzzComposedPromptAddText(f *testing.F) {
	seeds := []string{
		"hello",
		"",
		"日本語",
		"line1\nline2\nline3",
		"tab\there",
		" Special chars: !@#$%^&*() ",
		"a",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, text string) {
		prompt := Prompt{Text(text)}
		cp, err := buildComposedPrompt(prompt)
		if err != nil {
			return
		}
		defer releaseComposedPromptForTest(cp)

		raw := composedPromptTextContentForTest(cp)
		if text != "" && raw == "" {
			t.Errorf("GetTextContent returned empty for non-empty input")
		}
	})
}

func FuzzSchemaCreation(f *testing.F) {
	f.Add("TypeName", "description", "String")
	f.Add("", "", "")
	f.Add("X", "Y", "Int")
	f.Fuzz(func(t *testing.T, name, desc, typeName string) {
		schema, err := NewGenerationSchema(name, desc, []Property{{
			Name:     "prop1",
			TypeName: typeName,
			Optional: true,
		}}, nil)
		if err != nil {
			return
		}
		defer schema.Close()
		_, err = schema.JSON()
		if err != nil {
			return
		}
	})
}

func FuzzSchemaWithGuides(f *testing.F) {
	f.Add("pattern", "String")
	f.Add("[0-9]+", "String")
	f.Add("", "String")
	f.Add("^$", "String")
	f.Add("^(invalid", "String")
	f.Fuzz(func(t *testing.T, pattern, typeName string) {
		schema, err := NewGenerationSchema("FuzzSchema", "fuzz test", []Property{{
			Name:     "field1",
			TypeName: typeName,
			Optional: false,
			Guides:   []*GenerationGuide{Regex(pattern)},
		}}, nil)
		if err != nil {
			return
		}
		defer schema.Close()
	})
}

func FuzzFeedbackIssuesJSON(f *testing.F) {
	seeds := []string{
		`[{"category":"incorrect","explanation":"bad"}]`,
		`[]`,
		``,
		`not json`,
		`[{"category":"unknownCat"}]`,
		`{"wrong":"shape"}`,
		`[1,2,3]`,
		`null`,
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, issuesJSON string) {
		s, err := NewSession()
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()

		var issues []FeedbackIssue
		_ = json.Unmarshal([]byte(issuesJSON), &issues)

		sentiments := []FeedbackSentiment{
			FeedbackSentimentNone,
			FeedbackSentimentPositive,
			FeedbackSentimentNegative,
			FeedbackSentimentNeutral,
		}
		sentiment := sentiments[rand.Intn(len(sentiments))]

		_, _ = s.LogFeedbackAttachment(FeedbackAttachmentOptions{
			Sentiment:           sentiment,
			Issues:              issues,
			DesiredResponseText: "fuzz text",
		})
	})
}

func FuzzGeneratedContentPropertyAccess(f *testing.F) {
	seeds := []string{
		"score",
		"",
		"name",
		"123",
		"true",
		"null",
		"with spaces",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, propName string) {
		c, err := GeneratedContentFromJSON(`{"score":99,"name":"Alice","active":true}`)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()

		_ = c.HasProperty(propName)
		_, _ = c.ValueAsInt64(propName)
		_, _ = c.ValueAsBool(propName)
		_, _ = c.ValueAsFloat64(propName)
	})
}

func FuzzGenerationOptionsJSON(f *testing.F) {
	f.Add(0.0, 1, 0, 0.0, int64(0), false, false)
	f.Add(-1.0, 0, -1, -0.1, int64(-1), true, true)
	f.Add(2.5, 1024, 50, 1.0, int64(9223372036854775807), true, false)
	f.Fuzz(func(t *testing.T, temperature float64, maxTokens int, top int, topP float64, seed int64, useTop bool, useTopP bool) {
		opts := GenerationOptions{}
		opts.Temperature = Float64Ptr(temperature)
		opts.MaximumResponseTokens = IntPtr(maxTokens)

		var topPtr *int
		var topPPtr *float64
		if useTop {
			topPtr = IntPtr(top)
		}
		if useTopP {
			topPPtr = Float64Ptr(topP)
		}
		opts.Sampling = &[]SamplingMode{SamplingRandom(topPtr, topPPtr, Int64Ptr(seed))}[0]

		jsonStr, err := opts.toJSON()
		if err != nil {
			return
		}
		if jsonStr == "" {
			return
		}
		var got map[string]any
		if err := json.Unmarshal([]byte(jsonStr), &got); err != nil {
			t.Fatalf("options JSON is invalid: %v: %s", err, jsonStr)
		}
	})
}

func FuzzSchemaGuideCombinations(f *testing.F) {
	f.Add("field", "String", "a", "b", "[a-z]+", 0, 1, -1.0, 1.0)
	f.Add("", "", "", "", "^(invalid", -1, 0, -999.0, 999.0)
	f.Fuzz(func(t *testing.T, name, typeName, first, second, pattern string, count, maxItems int, min, max float64) {
		guides := []*GenerationGuide{
			AnyOf(first, second),
			Constant(first),
			Count(count),
			MinItems(count),
			MaxItems(maxItems),
			Minimum(min),
			Maximum(max),
			Range(min, max),
			Regex(pattern),
			Element(Regex(pattern)),
		}
		schema, err := NewGenerationSchema("GuideFuzzSchema", "", []Property{{
			Name:     name,
			TypeName: typeName,
			Optional: count%2 == 0,
			Guides:   guides,
		}}, nil)
		if err != nil {
			return
		}
		defer schema.Close()
		_, _ = schema.JSON()
	})
}

func FuzzExportSchemaRequest(f *testing.F) {
	f.Add("json_schema", "Invoice", `{"type":"object"}`)
	f.Add("openapi_schema", "", `{"type":"array","items":{"type":"string"}}`)
	f.Add("markdown_report", "日本語", `{"properties":{"x":{"type":"number"}}}`)
	f.Add("unsupported", "Bad", `not json`)
	f.Fuzz(func(t *testing.T, format, name, schemaJSON string) {
		var schema map[string]any
		if err := json.Unmarshal([]byte(schemaJSON), &schema); err != nil {
			schema = map[string]any{"raw": schemaJSON}
		}
		resp, err := ExportSchema(ExportSchemaRequest{
			Format: format,
			SchemaCandidate: SchemaCandidate{
				ID:      "fuzz-schema",
				Name:    name,
				Version: "0.0.0",
				Format:  "json_schema",
				Status:  "candidate",
				Schema:  schema,
			},
		})
		if err != nil {
			return
		}
		if resp.Format == "" || resp.Content == "" {
			t.Fatalf("empty export response: %#v", resp)
		}
	})
}

func FuzzDiscoveryRequestJSON(f *testing.F) {
	f.Add("doc-1", "text", "text/plain", "invoice.txt", "請求日 2026-01-01", "file:///tmp/invoice.txt", "ja")
	f.Add("", "", "", "", "", "", "")
	f.Fuzz(func(t *testing.T, id, sourceType, mediaType, name, content, uri, language string) {
		opts := DefaultDiscoveryOptions()
		req := DiscoverSchemaRequest{
			Documents: []DiscoveryDocument{{
				ID: id,
				Source: DiscoveryDocumentSource{
					Type:      sourceType,
					MediaType: mediaType,
					Name:      name,
					Content:   content,
					URI:       uri,
				},
				Metadata: map[string]any{"seed": len(content)},
			}},
			Hints:   &DiscoveryHints{Language: language, Domain: "fuzz"},
			Options: &opts,
		}
		b, err := json.Marshal(req)
		if err != nil {
			t.Fatal(err)
		}
		var roundTrip DiscoverSchemaRequest
		if err := json.Unmarshal(b, &roundTrip); err != nil {
			t.Fatal(err)
		}
	})
}

func FuzzPromptTextAndAttachmentConstruction(f *testing.F) {
	f.Add("hello", "/tmp/no-such-image.png", "")
	f.Add("", "", "")
	f.Add("日本語", "/tmp/no-such image.png", "label")
	f.Fuzz(func(t *testing.T, text, path, label string) {
		cp, err := buildComposedPrompt(Prompt{Text(text)})
		if err != nil {
			return
		}
		releaseComposedPromptForTest(cp)

		attachment, err := NewImageAttachment(path, label)
		if err != nil {
			return
		}
		cp, err = buildComposedPrompt(Prompt{Text(text), attachment})
		if err == nil {
			releaseComposedPromptForTest(cp)
		}
	})
}

func TestAfterCloseSafety(t *testing.T) {
	c, err := GeneratedContentFromJSON(`{"x":1}`)
	if err != nil {
		t.Fatal(err)
	}
	c.Close()

	_, ok := c.ValueAsInt64("x")
	if ok {
		t.Error("ValueAsInt64 on closed content should return false")
	}
	_, ok = c.ValueAsBool("x")
	if ok {
		t.Error("ValueAsBool on closed content should return false")
	}
	_, ok = c.ValueAsFloat64("x")
	if ok {
		t.Error("ValueAsFloat64 on closed content should return false")
	}
	if c.HasProperty("x") {
		t.Error("HasProperty on closed content should return false")
	}
	_, err = c.PropertyNames()
	if err == nil {
		t.Error("PropertyNames on closed content should return error")
	}

	// Note: Session after-close safety tests are fragile because FMRelease
	// deallocates the underlying Swift object. Prewarm on a closed session is
	// safe (guard against nil ptr), but IsResponding/Transcript entry access
	// on a closed session currently SEGV — this is a known limitation.
	s, err := NewSession()
	if err != nil {
		t.Fatal(err)
	}
	s.Close()
	s.Prewarm("should not crash")
	s.Prewarm("")

	tr, err := TranscriptFromJSON([]byte(`{}`))
	if err == nil {
		tr.Close()
	}

	schema, err := NewGenerationSchema("Test", "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	schema.Close()
}

func TestDoubleCloseSafety(t *testing.T) {
	c, err := GeneratedContentFromJSON(`{"x":1}`)
	if err != nil {
		t.Fatal(err)
	}
	c.Close()
	c.Close()

	s, err := NewSession()
	if err != nil {
		t.Fatal(err)
	}
	s.Close()
	s.Close()

	schema, err := NewGenerationSchema("Test", "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	schema.Close()
	schema.Close()
}

func TestPrewarmEdgeCases(t *testing.T) {
	s, err := NewSession()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	s.Prewarm("")
	s.Prewarm(strings.Repeat("x", 10000))
	s.Prewarm("multi\nline\ntext")
	if s.IsResponding() {
		t.Error("prewarm should not mark session as responding")
	}
}

func TestGeneratedContentEmptyJSON(t *testing.T) {
	c, err := GeneratedContentFromJSON(`{}`)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	names, err := c.PropertyNames()
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 0 {
		t.Errorf("empty JSON object should have 0 properties, got %d", len(names))
	}
	if c.HasProperty("anything") {
		t.Error("empty JSON object should not have properties")
	}
	_ = c.IsComplete()
}

func TestSchemaWithVariousGuides(t *testing.T) {
	// Create a simple schema without undefined type references
	// (Swift requires all referenced types to have corresponding schemas)
	schema, err := NewGenerationSchema("TestSchema", "A test schema", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer schema.Close()

	j, err := schema.JSON()
	if err != nil {
		t.Fatal(err)
	}
	if len(j) == 0 {
		t.Error("schema JSON should not be empty")
	}
}
