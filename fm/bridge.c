/*
 * Internal C bridge. Defines the trampolines that route Foundation Models C
 * callbacks back to Go-exported functions, and the per-slot tool callback
 * pool that lets a single Go //export dispatch tool calls per registered Tool.
 */

#include "bridge.h"

/* Forward declarations of //export'd Go functions defined in handles.go. */
extern void goSessionResponseCallback(int status, const char *content, size_t length, void *userInfo);
extern void goSessionStructuredCallback(int status, FMGeneratedContentRef content, void *userInfo);
extern void goToolCallbackSlot(int slot, FMGeneratedContentRef content, unsigned int callId);

FMTaskRef fm_session_respond(
    FMLanguageModelSessionRef session,
    FMComposedPrompt prompt,
    const char *optionsJSON,
    void *userInfo
) {
    return FMLanguageModelSessionRespond(session, prompt, optionsJSON, userInfo, goSessionResponseCallback);
}

FMTaskRef fm_session_respond_with_schema(
    FMLanguageModelSessionRef session,
    FMComposedPrompt prompt,
    FMGenerationSchemaRef schema,
    const char *optionsJSON,
    void *userInfo
) {
    return FMLanguageModelSessionRespondWithSchema(session, prompt, schema, optionsJSON, userInfo, goSessionStructuredCallback);
}

FMTaskRef fm_session_respond_with_schema_json(
    FMLanguageModelSessionRef session,
    FMComposedPrompt prompt,
    const char *schemaJSON,
    const char *optionsJSON,
    void *userInfo
) {
    return FMLanguageModelSessionRespondWithSchemaFromJSON(session, prompt, schemaJSON, optionsJSON, userInfo, goSessionStructuredCallback);
}

void fm_stream_iterate(
    FMLanguageModelSessionResponseStreamRef stream,
    void *userInfo
) {
    FMLanguageModelSessionResponseStreamIterate(stream, userInfo, goSessionResponseCallback);
}

/* --- Tool trampoline pool ------------------------------------------------- */

#define FM_TOOL_SLOTS 32

#define DEFINE_TOOL_TRAMPOLINE(N)                                             \
    static void fm_tool_trampoline_##N(FMGeneratedContentRef c, unsigned int id) { \
        goToolCallbackSlot(N, c, id);                                          \
    }

DEFINE_TOOL_TRAMPOLINE(0)
DEFINE_TOOL_TRAMPOLINE(1)
DEFINE_TOOL_TRAMPOLINE(2)
DEFINE_TOOL_TRAMPOLINE(3)
DEFINE_TOOL_TRAMPOLINE(4)
DEFINE_TOOL_TRAMPOLINE(5)
DEFINE_TOOL_TRAMPOLINE(6)
DEFINE_TOOL_TRAMPOLINE(7)
DEFINE_TOOL_TRAMPOLINE(8)
DEFINE_TOOL_TRAMPOLINE(9)
DEFINE_TOOL_TRAMPOLINE(10)
DEFINE_TOOL_TRAMPOLINE(11)
DEFINE_TOOL_TRAMPOLINE(12)
DEFINE_TOOL_TRAMPOLINE(13)
DEFINE_TOOL_TRAMPOLINE(14)
DEFINE_TOOL_TRAMPOLINE(15)
DEFINE_TOOL_TRAMPOLINE(16)
DEFINE_TOOL_TRAMPOLINE(17)
DEFINE_TOOL_TRAMPOLINE(18)
DEFINE_TOOL_TRAMPOLINE(19)
DEFINE_TOOL_TRAMPOLINE(20)
DEFINE_TOOL_TRAMPOLINE(21)
DEFINE_TOOL_TRAMPOLINE(22)
DEFINE_TOOL_TRAMPOLINE(23)
DEFINE_TOOL_TRAMPOLINE(24)
DEFINE_TOOL_TRAMPOLINE(25)
DEFINE_TOOL_TRAMPOLINE(26)
DEFINE_TOOL_TRAMPOLINE(27)
DEFINE_TOOL_TRAMPOLINE(28)
DEFINE_TOOL_TRAMPOLINE(29)
DEFINE_TOOL_TRAMPOLINE(30)
DEFINE_TOOL_TRAMPOLINE(31)

typedef void (*fm_tool_cb_t)(FMGeneratedContentRef, unsigned int);

static fm_tool_cb_t fm_tool_trampolines[FM_TOOL_SLOTS] = {
    fm_tool_trampoline_0,  fm_tool_trampoline_1,  fm_tool_trampoline_2,  fm_tool_trampoline_3,
    fm_tool_trampoline_4,  fm_tool_trampoline_5,  fm_tool_trampoline_6,  fm_tool_trampoline_7,
    fm_tool_trampoline_8,  fm_tool_trampoline_9,  fm_tool_trampoline_10, fm_tool_trampoline_11,
    fm_tool_trampoline_12, fm_tool_trampoline_13, fm_tool_trampoline_14, fm_tool_trampoline_15,
    fm_tool_trampoline_16, fm_tool_trampoline_17, fm_tool_trampoline_18, fm_tool_trampoline_19,
    fm_tool_trampoline_20, fm_tool_trampoline_21, fm_tool_trampoline_22, fm_tool_trampoline_23,
    fm_tool_trampoline_24, fm_tool_trampoline_25, fm_tool_trampoline_26, fm_tool_trampoline_27,
    fm_tool_trampoline_28, fm_tool_trampoline_29, fm_tool_trampoline_30, fm_tool_trampoline_31,
};

FMBridgedToolRef fm_tool_create_at_slot(
    int slot,
    const char *name,
    const char *description,
    FMGenerationSchemaRef parameters,
    int *outErrorCode,
    char **outErrorDescription
) {
    if (slot < 0 || slot >= FM_TOOL_SLOTS) {
        if (outErrorCode) *outErrorCode = 0xFF;
        return NULL;
    }
    return FMBridgedToolCreate(name, description, parameters, fm_tool_trampolines[slot], outErrorCode, outErrorDescription);
}
