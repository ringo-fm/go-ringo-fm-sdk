// Package fm provides Go bindings for Apple's Foundation Models framework,
// the on-device language model that powers Apple Intelligence on macOS 26+.
//
// This package wraps the FoundationModelsCBindings C library (Swift-backed)
// from the sibling python-apple-fm-sdk repository. To build, the C dylib must
// be available at link time. The standard workflow:
//
//  1. Build the dylib in the sibling repo:
//     cd ../python-apple-fm-sdk/foundation-models-c && swift build -c release
//
//  2. Point cgo at the build output before running go build:
//     export FM_LIB_DIR="$(pwd)/.build/release"
//     export CGO_LDFLAGS="-L${FM_LIB_DIR} -Wl,-rpath,${FM_LIB_DIR}"
//     go build ./...
//
// The header is vendored under internal/fmlib/include so the build is
// header-self-contained; only the dylib path is external.
package fm

/*
#cgo CFLAGS: -I${SRCDIR}/../internal/fmlib/include -I${SRCDIR}
#cgo LDFLAGS: -lFoundationModels

#include "bridge.h"
*/
import "C"
