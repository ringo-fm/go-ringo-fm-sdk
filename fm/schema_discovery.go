package fm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type DiscoverSchemaRequest struct {
	Documents      []DiscoveryDocument `json:"documents"`
	Hints          *DiscoveryHints     `json:"hints,omitempty"`
	Options        *DiscoveryOptions   `json:"options,omitempty"`
	ExistingSchema map[string]any      `json:"existing_schema,omitempty"`
}

type DiscoveryDocument struct {
	ID       string                  `json:"id"`
	Source   DiscoveryDocumentSource `json:"source"`
	Metadata map[string]any          `json:"metadata,omitempty"`
}

type DiscoveryDocumentSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type,omitempty"`
	Name      string `json:"name,omitempty"`
	Content   string `json:"content,omitempty"`
	URI       string `json:"uri,omitempty"`
}

type DiscoveryHints struct {
	Language             string `json:"language,omitempty"`
	Domain               string `json:"domain,omitempty"`
	DocumentType         string `json:"document_type,omitempty"`
	ExpectedSchemaFormat string `json:"expected_schema_format,omitempty"`
}

type DiscoveryOptions struct {
	IncludeEvidence            bool    `json:"include_evidence"`
	IncludeLayout              bool    `json:"include_layout"`
	IncludeRawExtractions      bool    `json:"include_raw_extractions"`
	MinPresenceRateForRequired float64 `json:"min_presence_rate_for_required"`
	MinConfidence              float64 `json:"min_confidence"`
	MaxFieldCandidates         int     `json:"max_field_candidates"`
	MergeSimilarFields         bool    `json:"merge_similar_fields"`
	InferConstraints           bool    `json:"infer_constraints"`
	InferArrays                bool    `json:"infer_arrays"`
	InferNestedObjects         bool    `json:"infer_nested_objects"`
}

func DefaultDiscoveryOptions() DiscoveryOptions {
	return DiscoveryOptions{
		IncludeEvidence:            true,
		IncludeLayout:              true,
		IncludeRawExtractions:      false,
		MinPresenceRateForRequired: 0.8,
		MinConfidence:              0.5,
		MaxFieldCandidates:         200,
		MergeSimilarFields:         true,
		InferConstraints:           true,
		InferArrays:                true,
		InferNestedObjects:         true,
	}
}

type DiscoverSchemaResponse struct {
	SchemaCandidate   SchemaCandidate     `json:"schema_candidate"`
	FieldCandidates   []FieldCandidate    `json:"field_candidates"`
	DocumentSummaries []map[string]any    `json:"document_summaries"`
	Conflicts         []DiscoveryConflict `json:"conflicts"`
	Warnings          []DiscoveryWarning  `json:"warnings"`
	ReviewFindings    []ReviewFinding     `json:"review_findings"`
	Metrics           DiscoveryMetrics    `json:"metrics"`
	SchemaDiff        *SchemaDiff         `json:"schema_diff,omitempty"`
}

type SchemaCandidate struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Version  string         `json:"version"`
	Format   string         `json:"format"`
	Status   string         `json:"status"`
	Schema   map[string]any `json:"schema"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type FieldCandidate struct {
	ID                   string          `json:"id"`
	CanonicalName        string          `json:"canonical_name"`
	DisplayName          string          `json:"display_name"`
	Path                 string          `json:"path"`
	TypeCandidates       []TypeCandidate `json:"type_candidates"`
	Labels               []string        `json:"labels"`
	Presence             Presence        `json:"presence"`
	RequiredCandidate    bool            `json:"required_candidate"`
	ArrayCandidate       bool            `json:"array_candidate"`
	NullableCandidate    bool            `json:"nullable_candidate"`
	Examples             []string        `json:"examples"`
	Evidence             []Evidence      `json:"evidence"`
	Confidence           float64         `json:"confidence"`
	ReviewRequired       bool            `json:"review_required"`
	SuggestedConstraints map[string]any  `json:"suggested_constraints,omitempty"`
}

type TypeCandidate struct {
	Type       string  `json:"type"`
	Format     string  `json:"format,omitempty"`
	Confidence float64 `json:"confidence"`
}

type Presence struct {
	DocumentCount int     `json:"document_count"`
	PresentCount  int     `json:"present_count"`
	PresenceRate  float64 `json:"presence_rate"`
}

type Evidence struct {
	DocumentID  string       `json:"document_id"`
	Page        *int         `json:"page,omitempty"`
	Text        string       `json:"text,omitempty"`
	LabelText   string       `json:"label_text,omitempty"`
	ValueText   string       `json:"value_text,omitempty"`
	BoundingBox *BoundingBox `json:"bounding_box,omitempty"`
	Confidence  float64      `json:"confidence"`
	Reason      string       `json:"reason"`
}

type BoundingBox struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type DiscoveryConflict struct {
	ID                string   `json:"id"`
	Type              string   `json:"type"`
	Severity          string   `json:"severity"`
	Message           string   `json:"message"`
	FieldCandidateIDs []string `json:"field_candidate_ids"`
	SuggestedAction   string   `json:"suggested_action,omitempty"`
}

type DiscoveryWarning struct {
	ID       string `json:"id"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Code     string `json:"code"`
}

type ReviewFinding struct {
	ID                string  `json:"id"`
	Target            string  `json:"target"`
	Reason            string  `json:"reason"`
	Description       string  `json:"description"`
	SuggestedDecision string  `json:"suggested_decision"`
	Confidence        float64 `json:"confidence"`
}

type DiscoveryMetrics struct {
	DocumentCount           int     `json:"document_count"`
	FieldCandidateCount     int     `json:"field_candidate_count"`
	AverageConfidence       float64 `json:"average_confidence"`
	LowConfidenceFieldCount int     `json:"low_confidence_field_count"`
	ConflictCount           int     `json:"conflict_count"`
	ReviewRequiredCount     int     `json:"review_required_count"`
	ExtractionSuccessRate   float64 `json:"extraction_success_rate"`
	SchemaCoverageRate      float64 `json:"schema_coverage_rate"`
}

type SchemaDiff struct {
	AddedFields            []map[string]any `json:"added_fields"`
	RemovedFields          []map[string]any `json:"removed_fields"`
	ChangedFields          []map[string]any `json:"changed_fields"`
	RenamedFieldCandidates []map[string]any `json:"renamed_field_candidates"`
	AliasChanges           []map[string]any `json:"alias_changes"`
}

type ExportSchemaRequest struct {
	SchemaCandidate SchemaCandidate      `json:"schema_candidate"`
	Format          string               `json:"format"`
	Options         *ExportSchemaOptions `json:"options,omitempty"`
}

type ExportSchemaOptions struct {
	IncludeExtensions   bool `json:"include_extensions"`
	IncludeDescriptions bool `json:"include_descriptions"`
	IncludeExamples     bool `json:"include_examples"`
}

type ExportSchemaResponse struct {
	Format  string `json:"format"`
	Content string `json:"content"`
}

func (s *Session) DiscoverSchema(ctx context.Context, request DiscoverSchemaRequest, opts ...RespondOption) (*DiscoverSchemaResponse, error) {
	prompt, err := buildDiscoveryPrompt(request)
	if err != nil {
		return nil, err
	}
	content, err := s.RespondWithJSONSchema(ctx, TextPrompt(prompt), []byte(discoveryResponseSchemaJSON), opts...)
	if err != nil {
		return nil, err
	}
	defer content.Close()
	var response DiscoverSchemaResponse
	if err := content.Decode(&response); err != nil {
		return nil, err
	}
	return &response, nil
}

func ExportSchema(request ExportSchemaRequest) (*ExportSchemaResponse, error) {
	switch request.Format {
	case "", "json_schema", "openapi_schema":
		b, err := json.MarshalIndent(request.SchemaCandidate.Schema, "", "  ")
		if err != nil {
			return nil, err
		}
		format := request.Format
		if format == "" {
			format = "json_schema"
		}
		return &ExportSchemaResponse{Format: format, Content: string(b)}, nil
	case "markdown_report":
		return &ExportSchemaResponse{Format: request.Format, Content: markdownSchemaReport(request.SchemaCandidate)}, nil
	default:
		return nil, fmt.Errorf("ExportSchema: unsupported format %q", request.Format)
	}
}

func buildDiscoveryPrompt(request DiscoverSchemaRequest) (string, error) {
	b, err := json.MarshalIndent(request, "", "  ")
	if err != nil {
		return "", err
	}
	return "Discover a reusable schema candidate from the provided documents.\n" +
		"Treat every result as a candidate, not an approved schema.\n" +
		"Return only data that matches the supplied JSON Schema.\n" +
		"Preserve evidence only when requested. Mark ambiguous or low-confidence fields for review.\n" +
		"Do not add raw document text to warnings or logs unless it is evidence requested by the caller.\n\n" +
		"Request JSON:\n" + string(b), nil
}

func markdownSchemaReport(candidate SchemaCandidate) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Schema Candidate: %s\n\n", candidate.Name)
	fmt.Fprintf(&b, "- ID: %s\n", candidate.ID)
	fmt.Fprintf(&b, "- Version: %s\n", candidate.Version)
	fmt.Fprintf(&b, "- Format: %s\n", candidate.Format)
	fmt.Fprintf(&b, "- Status: %s\n", candidate.Status)
	return b.String()
}

const discoveryResponseSchemaJSON = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "additionalProperties": false,
  "required": ["schema_candidate", "field_candidates", "document_summaries", "conflicts", "warnings", "review_findings", "metrics"],
  "properties": {
    "schema_candidate": {"type": "object"},
    "field_candidates": {"type": "array", "items": {"type": "object"}},
    "document_summaries": {"type": "array", "items": {"type": "object"}},
    "conflicts": {"type": "array", "items": {"type": "object"}},
    "warnings": {"type": "array", "items": {"type": "object"}},
    "review_findings": {"type": "array", "items": {"type": "object"}},
    "metrics": {"type": "object"},
    "schema_diff": {"type": "object"}
  }
}`
