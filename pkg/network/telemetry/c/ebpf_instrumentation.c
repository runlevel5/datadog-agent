#include "bpf_telemetry.h"
#include "bpf_helpers.h"
#include "map-defs.h"
#include "compiler.h"

#define FETCH_TELEMETRY_BLOB() ({ \
    instrumentation_blob_t *__tb; \
    asm("%0 = *(u64 *)(r10 - 512)" : "=r"(__tb)); \
    __tb; \
})

SEC("ebpf_instrumentation/trampoline_handler")
int ebpf_instrumentation__trampoline_handler() {
    u64 key = 0;
    instrumentation_blob_t* tb = bpf_map_lookup_elem(&bpf_telemetry_map, &key);
    if (tb == NULL) {
        asm ("r2 = 0");
        asm("*(u64 *)(r10 - 512) = r2");
        return 1;
    }

    // Cache telemetry blob on stack
    asm ("*(u64 *)(r10 - 512) = r0");
    return 1;
}

SEC("ebpf_instrumentation/map_error_telemetry")
unsigned long ebpf_instrumentation__map_error_telemetry(unsigned long callsite, long error, u64 map_index) {
    instrumentation_blob_t *tb = FETCH_TELEMETRY_BLOB();
    asm ("*(u64 *)(r10 - 504) = r1");
    if (tb == NULL)
        return callsite;

    error = error * -1;
    if (error >= T_MAX_ERRNO) {
        error = T_MAX_ERRNO - 1;
    }
    error &= T_MAX_ERRNO;
    __sync_fetch_and_add(&tb->map_err_telemetry[map_index].err_count[error], 1);

    return callsite;
}

SEC("ebpf_instrumentation/helper_error_telemetry")
unsigned long ebpf_instrumentation__helper_error_telemetry(unsigned long callsite, long error, unsigned int helper_index) {
    instrumentation_blob_t *tb = FETCH_TELEMETRY_BLOB();
    asm ("*(u64 *)(r10 - 504) = r1");
    if (tb == NULL)
        return callsite;

    tb->telemetry_active = 2;
    u64 program_index = 0;
    LOAD_CONSTANT("telemetry_program_id_key", program_index);

    error = error * -1;
    if (error >= T_MAX_ERRNO) {
        error = T_MAX_ERRNO - 1;
    }
    error &= T_MAX_ERRNO;
    if (helper_index >= 0) {
        __sync_fetch_and_add(&tb->helper_err_telemetry[program_index].err_count[(helper_index * T_MAX_ERRNO) + error], 1);
    }

    return callsite;
}
