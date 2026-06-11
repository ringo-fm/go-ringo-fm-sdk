# Foundation Models SDK for Go

Go bindings for Apple's [Foundation Models framework](https://developer.apple.com/documentation/foundationmodels) — the on-device language model that powers Apple Intelligence on macOS 26+.

This repository vendors the `foundation-models-c` Swift/C bridge source locally, so builds do not depend on a sibling checkout of `python-apple-fm-sdk`.

## Requirements

- macOS 26+ on Apple Silicon with Apple Intelligence enabled
- Xcode 26+
- Go 1.22+
- Swift Package Manager

## Build

```sh
# 1. Build the vendored Swift C bindings once.
cd foundation-models-c
swift build -c release

# 2. Tell cgo where to find the dylib.
export FM_LIB_DIR="$(pwd)/.build/release"
export CGO_LDFLAGS="-L${FM_LIB_DIR} -Wl,-rpath,${FM_LIB_DIR}"

# 3. Build / run examples from the Go repo.
cd ..
go build ./...
go run ./examples/simple
go run ./examples/streaming
go run ./examples/transcript
```

The bridge source lives under `foundation-models-c/`. The C header used by cgo is vendored under `internal/fmlib/include/FoundationModels.h`.

## Quick start

```go
package main

import (
	"context"
	"fmt"

	fm "github.com/f4ah6o/go-ringo-fm-sdk/fm"
)

func main() {
	model := fm.NewSystemLanguageModel()
	defer model.Close()
	if ok, reason := model.IsAvailable(); !ok {
		fmt.Println("unavailable:", reason)
		return
	}

	session, _ := fm.NewSession(fm.WithInstructions("Be concise."))
	defer session.Close()

	out, _ := session.Respond(context.Background(), fm.TextPrompt("Hello!"))
	fmt.Println(out)
}
```

### Guided generation via struct tags

The Python decorator `@fm.generable` is replaced by `fm` struct tags read with reflection.

```go
type Cat struct {
	Name string `fm:"name,description=Cat's name"`
	Age  int    `fm:"age,description=Age in years,range=0:20"`
	Food string `fm:"food,anyOf=fish|chicken|tuna"`
}

var cat Cat
err := session.RespondInto(ctx, fm.TextPrompt("Make me a cat"), &cat)
```

Tag keys: `description`, `anyOf` (|-separated), `constant`, `count`, `minItems`, `maxItems`, `minimum`, `maximum`, `range` (`min:max`), `regex`. Pointer fields are treated as optional.

### Streaming

```go
snapshots, errs := session.StreamResponse(ctx, fm.TextPrompt("Tell a story"))
prev := ""
for snap := range snapshots {
	fmt.Print(snap[len(prev):])
	prev = snap
}
if err := <-errs; err != nil { /* handle */ }
```

Snapshots are cumulative (each value contains the full text so far) — the same shape as the Python SDK.

### Tools

```go
type calc struct{ schema *fm.GenerationSchema }

func (c *calc) Name() string                                       { return "add" }
func (c *calc) Description() string                                { return "Adds two numbers." }
func (c *calc) ArgumentsSchema() *fm.GenerationSchema              { return c.schema }
func (c *calc) Call(ctx context.Context, args *fm.GeneratedContent) (string, error) {
	m, _ := args.AsMap()
	return fmt.Sprintf("%v", m["a"].(float64)+m["b"].(float64)), nil
}

addSchema, _ := fm.SchemaFor[struct {
	A float64 `fm:"a"`
	B float64 `fm:"b"`
}]()
bt, _ := fm.RegisterTool(&calc{schema: addSchema})
defer bt.Close()

session, _ := fm.NewSession(fm.WithTools(bt))
```

Up to 32 tools may be registered simultaneously across the process (raise `FM_TOOL_SLOTS` in `fm/cgo.go` if you need more).

### Schema Discovery

`Session.DiscoverSchema` uses guided generation to infer a reviewable schema
candidate from text, JSON, or already-extracted documents. The response includes
field candidates, evidence, warnings, metrics, and review findings; treat it as
a draft for human review, not an approved schema.

```go
opts := fm.DefaultDiscoveryOptions()
response, err := session.DiscoverSchema(ctx, fm.DiscoverSchemaRequest{
	Documents: []fm.DiscoveryDocument{{
		ID: "doc-1",
		Source: fm.DiscoveryDocumentSource{
			Type:    "text",
			Content: "請求日 2026-01-01\n合計 12,000円",
		},
	}},
	Options: &opts,
})
```

## Status

Alpha. The Go API is not yet stable and macOS 26 is itself in early-adopter territory.

## License

See `LICENSE.md`.

This repository includes vendored source under `foundation-models-c/` derived from Apple's `python-apple-fm-sdk` project and keeps the upstream license headers intact.
