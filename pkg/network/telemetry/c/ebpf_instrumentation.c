#include "bpf_telemetry.h"
#include "bpf_helpers.h"
#include "map-defs.h"

#define FUNC_INFO_METADATA_SINK() { asm("r1 = r1"); }

SEC("ebpf_telemetry/trampoline_handler")
void ebpf_telemetry__trampoline_handler() {
    FUNC_INFO_METADATA_SINK();
    u64 key = 0;
    bpf_map_lookup_elem(&helper_err_telemetry_map, &key);
    asm ("*(u64 *)(r10 - 512) = r0");
    return;
}
