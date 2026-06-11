package fm

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// SamplingMode controls how tokens are chosen from the model's probability
// distribution. Use SamplingGreedy or SamplingRandom to construct one.
type SamplingMode struct {
	mode                 string // "greedy" or "random"
	top                  *int
	probabilityThreshold *float64
	seed                 *int64
}

// SamplingGreedy picks the most-likely token at every step.
func SamplingGreedy() SamplingMode { return SamplingMode{mode: "greedy"} }

// SamplingRandom samples randomly with optional constraints. Pass nil for any
// unused parameter. Top and ProbabilityThreshold are mutually exclusive; if
// both are non-nil SamplingRandom returns an error in GenerationOptions.Validate.
func SamplingRandom(top *int, probabilityThreshold *float64, seed *int64) SamplingMode {
	return SamplingMode{mode: "random", top: top, probabilityThreshold: probabilityThreshold, seed: seed}
}

// IntPtr / Float64Ptr / Int64Ptr help build SamplingRandom calls.
func IntPtr(v int) *int             { return &v }
func Float64Ptr(v float64) *float64 { return &v }
func Int64Ptr(v int64) *int64       { return &v }

// GenerationOptions tunes a single Respond/Stream call. The zero value uses
// model defaults.
type GenerationOptions struct {
	Sampling              *SamplingMode
	Temperature           *float64
	MaximumResponseTokens *int
}

// Validate checks the option combinations.
func (g GenerationOptions) Validate() error {
	if g.Sampling != nil && g.Sampling.mode == "random" {
		if g.Sampling.top != nil && g.Sampling.probabilityThreshold != nil {
			return fmt.Errorf("generation options: sampling cannot set both top and probabilityThreshold")
		}
		if g.Sampling.top != nil && *g.Sampling.top <= 0 {
			return fmt.Errorf("generation options: sampling.top must be positive")
		}
		if g.Sampling.probabilityThreshold != nil {
			p := *g.Sampling.probabilityThreshold
			if p < 0.0 || p > 1.0 {
				return fmt.Errorf("generation options: sampling.probabilityThreshold must be in [0, 1]")
			}
		}
	}
	if g.Temperature != nil && *g.Temperature < 0.0 {
		return fmt.Errorf("generation options: temperature must be non-negative")
	}
	if g.MaximumResponseTokens != nil && *g.MaximumResponseTokens <= 0 {
		return fmt.Errorf("generation options: maximumResponseTokens must be positive")
	}
	return nil
}

// toJSON serializes the options into the JSON shape the C bindings expect.
// Mirrors GenerationOptions.to_dict() in the Python SDK. Returns ("", nil)
// when no fields are set.
func (g GenerationOptions) toJSON() (string, error) {
	if err := g.Validate(); err != nil {
		return "", err
	}
	m := map[string]any{}
	if g.Sampling != nil {
		sm := map[string]any{"mode": g.Sampling.mode}
		if g.Sampling.mode == "random" {
			if g.Sampling.top != nil {
				sm["top_k"] = strconv.Itoa(*g.Sampling.top)
			}
			if g.Sampling.probabilityThreshold != nil {
				sm["top_p"] = strconv.FormatFloat(*g.Sampling.probabilityThreshold, 'f', -1, 64)
			}
			if g.Sampling.seed != nil {
				sm["seed"] = strconv.FormatInt(*g.Sampling.seed, 10)
			}
		}
		m["sampling"] = sm
	}
	if g.Temperature != nil {
		m["temperature"] = *g.Temperature
	}
	if g.MaximumResponseTokens != nil {
		m["maximum_response_tokens"] = *g.MaximumResponseTokens
	}
	if len(m) == 0 {
		return "", nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
