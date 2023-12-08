#include "bpf_telemetry.h"
#include "bpf_helpers.h"
#include "map-defs.h"

#define FETCH_TELEMETRY_BLOB() ({ \
    instrumentation_blob_t *__tb; \
    asm("%0 = *(u64 *)(r10 - 512)" : "=r"(__tb)); \
    __tb; \
})

SEC("ebpf_instrumentation/trampoline_handler")
void ebpf_instrumentation__trampoline_handler() {
    u64 key = 0;
    instrumentation_blob_t* tb = bpf_map_lookup_elem(&bpf_telemetry_map, &key);
    if (tb == NULL)
        return;

    // Cache telemetry blob on stack
    asm ("*(u64 *)(r10 - 512) = r0");
    tb->telemetry_active = 1;
    return;
}

SEC("ebpf_instrumentation/map_error_telemetry")
void ebpf_instrumentation__map_error_telemetry(unsigned long callsite, long error, u64 map_index) {
    instrumentation_blob_t *tb = FETCH_TELEMETRY_BLOB();
    error = error * -1;
    if (error >= T_MAX_ERRNO) {
        error = T_MAX_ERRNO - 1;
        error &= (T_MAX_ERRNO - 1);
    }
    error &= (T_MAX_ERRNO - 1);
    __sync_fetch_and_add(&tb->map_err_telemetry[map_index].err_count[error], 1);

    return;
}
