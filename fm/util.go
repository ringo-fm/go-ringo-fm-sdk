package fm

/*
#include <stdlib.h>
#include "FoundationModels.h"
*/
import "C"

// goStringAndFree converts a C string allocated by the FM library to a Go
// string and frees the original. Safe to call with a nil pointer.
func goStringAndFree(s *C.char) string {
	if s == nil {
		return ""
	}
	out := C.GoString(s)
	C.FMFreeString(s)
	return out
}
