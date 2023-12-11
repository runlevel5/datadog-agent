#ifndef BPF_TELEMETRY_H
#define BPF_TELEMETRY_H

#include "bpf_helpers.h"
#include "telemetry_types.h"
#include "map-defs.h"

BPF_ARRAY_MAP(bpf_telemetry_map, instrumentation_blob_t, 1);

#define STR(x) #x
#define MK_MAP_INDX(key) STR(key##_telemetry_key)

BPF_HASH_MAP(helper_err_telemetry_map, unsigned long, helper_err_telemetry_t, 256)

#define PATCH_TARGET_MAP_ERRORS -2
static void *(*bpf_telemetry_map_errors_patch)(unsigned long callsite, long error, unsigned long map_index) = (void *)PATCH_TARGET_MAP_ERRORS;

#define map_update_with_telemetry(fn, map, args...)                                \
    ({                                                                             \
        long errno_ret;                                                            \
        errno_ret = fn(&map, args);                                                \
        unsigned long ret, map_index;                                              \
        LOAD_CONSTANT(MK_MAP_INDX(map), map_index);                                \
        LOAD_CONSTANT("retpoline_jump_addr", ret);                                 \
        if (errno_ret < 0) {                                                       \
            bpf_telemetry_map_errors_patch(ret, errno_ret, map_index);             \
        }                                                                          \
        errno_ret;                                                                 \
    })


//#define map_update_with_telemetry(fn, map, args...)                                \
//    ({                                                                             \
//        long errno_ret, errno_slot;                                                \
//        errno_ret = fn(&map, args);                                                \
//        unsigned long err_telemetry_key;                                           \
//        LOAD_CONSTANT(MK_MAP_INDX(map), err_telemetry_key);                             \
//        if (errno_ret < 0 && err_telemetry_key > 0) {                              \
//            map_err_telemetry_t *entry =                                           \
//                bpf_map_lookup_elem(&map_err_telemetry_map, &err_telemetry_key);   \
//            if (entry) {                                                           \
//                errno_slot = errno_ret * -1;                                       \
//                if (errno_slot >= T_MAX_ERRNO) {                                   \
//                    errno_slot = T_MAX_ERRNO - 1;                                  \
//                    errno_slot &= (T_MAX_ERRNO - 1);                               \
//                }                                                                  \
//                errno_slot &= (T_MAX_ERRNO - 1);                                   \
//                long *target = &entry->err_count[errno_slot];                      \
//                unsigned long add = 1;                                             \
//                /* Patched instruction for 4.14+: __sync_fetch_and_add(target, 1);
//                 * This patch point is placed here because the above instruction
//                 * fails on the 4.4 verifier. On 4.4 this instruction is replaced
//                 * with a nop: r1 = r1 */                                          \
//                bpf_telemetry_update_patch((unsigned long)target, add);            \
//            }                                                                      \
//        }                                                                          \
//        errno_ret;                                                                 \
//    })

#define MK_FN_INDX(fn) FN_INDX_##fn

#define FN_INDX_bpf_probe_read read_indx

#define FN_INDX_bpf_probe_read_kernel read_kernel_indx
#define FN_INDX_bpf_probe_read_kernel_str read_kernel_indx

#define FN_INDX_bpf_probe_read_user read_user_indx
#define FN_INDX_bpf_probe_read_user_str read_user_indx

#define FN_INDX_bpf_skb_load_bytes skb_load_bytes
#define FN_INDX_bpf_perf_event_output perf_event_output

#define helper_with_telemetry(fn, ...)                                                          \
    ({                                                                                          \
        long errno_ret = fn(__VA_ARGS__);                                                       \
        errno_ret;                                                                              \
    })



//#define helper_with_telemetry(fn, ...)                                                          \
//    ({                                                                                          \
//        long errno_ret = fn(__VA_ARGS__);                                                       \
//        unsigned long telemetry_program_id;                                                     \
//        LOAD_CONSTANT("telemetry_program_id_key", telemetry_program_id);                        \
//        if (errno_ret < 0 && telemetry_program_id > 0) {                                        \
//            helper_err_telemetry_t *entry =                                                     \
//                bpf_map_lookup_elem(&helper_err_telemetry_map, &telemetry_program_id);          \
//            if (entry) {                                                                        \
//                helper_indx = MK_FN_INDX(fn);                                                   \
//                errno_slot = errno_ret * -1;                                                    \
//                if (errno_slot >= T_MAX_ERRNO) {                                                \
//                    errno_slot = T_MAX_ERRNO - 1;                                               \
//                    /* This is duplicated below because on clang 14.0.6 the compiler
//                     * concludes that this if-check will always force errno_slot in range
//                     * (0, T_MAX_ERRNO-1], and removes the bounds check, causing the verifier
//                     * to trip. Duplicating this check forces clang not to omit the check */    \
//                    errno_slot &= (T_MAX_ERRNO - 1);                                            \
//                }                                                                               \
//                errno_slot &= (T_MAX_ERRNO - 1);                                                \
//                if (helper_indx >= 0) {                                                         \
//                    long *target = &entry->err_count[(helper_indx * T_MAX_ERRNO) + errno_slot]; \
//                    unsigned long add = 1;                                                      \
//                    /* Patched instruction for 4.14+: __sync_fetch_and_add(target, 1);
//                     * This patch point is placed here because the above instruction
//                     * fails on the 4.4 verifier. On 4.4 this instruction is replaced
//                     * with a nop: r1 = r1 */                                                   \
//                    bpf_telemetry_update_patch((unsigned long)target, add);                     \
//                }                                                                               \
//            }                                                                                   \
//        }                                                                                       \
//
//        errno_ret;                                                                              \
//    })

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
