#ifndef BPF_TELEMETRY_H
#define BPF_TELEMETRY_H

#include "bpf_helpers.h"
#include "telemetry_types.h"
#include "map-defs.h"

BPF_ARRAY_MAP(bpf_telemetry_map, instrumentation_blob_t, 1);

#define PATCH_TARGET_MAP_ERRORS -2
#define PATCH_TARGET_HELPER_ERRORS -3
static void *(*bpf_telemetry_map_errors_patch)(unsigned long callsite, long error, unsigned long map_index) = (void *)PATCH_TARGET_MAP_ERRORS;
static void *(*bpf_telemetry_helper_errors_patch)(unsigned long callsite, long error, unsigned int helper_index) = (void *)PATCH_TARGET_HELPER_ERRORS;

#define STR(x) #x
#define MK_MAP_INDX(key) STR(key##_telemetry_key)

#define map_update_with_telemetry(fn, map, args...)                    \
    ({                                                                 \
        long errno_ret;                                                \
        errno_ret = fn(&map, args);                                    \
        unsigned long ret, map_index;                                  \
        LOAD_CONSTANT(MK_MAP_INDX(map), map_index);                    \
        LOAD_CONSTANT("retpoline_jump_addr", ret);                     \
        if (errno_ret < 0) {                                           \
            bpf_telemetry_map_errors_patch(ret, errno_ret, map_index); \
        }                                                              \
        errno_ret;                                                     \
    })

#define MK_FN_INDX(fn) FN_INDX_##fn

#define FN_INDX_bpf_probe_read read_indx
#define FN_INDX_bpf_probe_read_kernel read_kernel_indx
#define FN_INDX_bpf_probe_read_kernel_str read_kernel_indx
#define FN_INDX_bpf_probe_read_user read_user_indx
#define FN_INDX_bpf_probe_read_user_str read_user_indx
#define FN_INDX_bpf_skb_load_bytes skb_load_bytes
#define FN_INDX_bpf_perf_event_output perf_event_output

#define helper_with_telemetry(fn, ...)                                                                              \
    ({                                                                                                              \
        long errno_ret = fn(__VA_ARGS__);                                                                  \
        if (errno_ret < 0) {                                                                                  \
            unsigned long ret;                                                                                       \
            LOAD_CONSTANT("retpoline_jump_addr", ret);                                                              \
            /* We pack two parameters in a single u64 to minimize the stack space used on caller-saved registers */ \
            bpf_telemetry_helper_errors_patch(ret, errno_ret, MK_FN_INDX(fn));                                                      \
        }                                                                                                           \
        errno_ret;                                                                                          \
    })

#define bpf_map_update_with_telemetry(map, key, val, flags) \
    map_update_with_telemetry(bpf_map_update_elem, map, key, val, flags)

#define bpf_probe_read_with_telemetry(...) \
    helper_with_telemetry(bpf_probe_read, __VA_ARGS__)

#define bpf_probe_read_str_with_telemetry(...) \
    helper_with_telemetry(bpf_probe_read_str, __VA_ARGS__)

#define bpf_probe_read_user_with_telemetry(...) \
    helper_with_telemetry(bpf_probe_read_user, __VA_ARGS__)

#define bpf_probe_read_user_str_with_telemetry(...) \
    helper_with_telemetry(bpf_probe_read_user_str, __VA_ARGS__)

#define bpf_probe_read_kernel_with_telemetry(...) \
    helper_with_telemetry(bpf_probe_read_kernel, __VA_ARGS__)

#define bpf_probe_read_kernel_str_with_telemetry(...) \
    helper_with_telemetry(bpf_probe_read_kernel_str, __VA_ARGS__)

#define bpf_skb_load_bytes_with_telemetry(...) \
    helper_with_telemetry(bpf_skb_load_bytes, __VA_ARGS__)

#define bpf_perf_event_output_with_telemetry(...) \
    helper_with_telemetry(bpf_perf_event_output, __VA_ARGS__)

#endif // BPF_TELEMETRY_H
