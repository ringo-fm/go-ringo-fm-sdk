/*
 * Internal C bridge: forward declarations for trampolines and helpers
 * implemented in bridge.c. Kept separate from cgo.go because cgo forbids
 * C-function definitions in the preamble of a package that uses //export.
 */

#ifndef FM_GO_BRIDGE_H
#define FM_GO_BRIDGE_H

#include "FoundationModels.h"

/* Trampoline helpers that hand our exported Go callbacks to the C API. */
FMTaskRef fm_session_respond(
    FMLanguageModelSessionRef session,
    FMComposedPrompt prompt,
    const char *optionsJSON,
    void *userInfo
);

FMTaskRef fm_session_respond_with_schema(
    FMLanguageModelSessionRef session,
    FMComposedPrompt prompt,
    FMGenerationSchemaRef schema,
    const char *optionsJSON,
    void *userInfo
);

FMTaskRef fm_session_respond_with_schema_json(
    FMLanguageModelSessionRef session,
    FMComposedPrompt prompt,
    const char *schemaJSON,
    const char *optionsJSON,
    void *userInfo
);

void fm_stream_iterate(
    FMLanguageModelSessionResponseStreamRef stream,
    void *userInfo
);

/* Tool callback trampoline pool. Pool size cap is FM_TOOL_SLOTS (32). */
FMBridgedToolRef fm_tool_create_at_slot(
    int slot,
    const char *name,
    const char *description,
    FMGenerationSchemaRef parameters,
    int *outErrorCode,
    char **outErrorDescription
);

#endif /* FM_GO_BRIDGE_H */
