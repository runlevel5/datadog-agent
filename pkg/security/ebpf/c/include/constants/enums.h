#ifndef _CONSTANTS_ENUMS_H
#define _CONSTANTS_ENUMS_H

enum event_type {
    EVENT_ANY = 0,
    EVENT_FIRST_DISCARDER = 1,
    EVENT_OPEN = EVENT_FIRST_DISCARDER,
    EVENT_MKDIR,
    EVENT_LINK,
    EVENT_RENAME,
    EVENT_UNLINK,
    EVENT_RMDIR,
    EVENT_CHMOD,
    EVENT_CHOWN,
    EVENT_UTIME,
    EVENT_SETXATTR,
    EVENT_REMOVEXATTR,
    EVENT_CHDIR,
    EVENT_LAST_DISCARDER = EVENT_CHDIR,

    EVENT_MOUNT,
    EVENT_UMOUNT,
    EVENT_FORK,
    EVENT_EXEC,
    EVENT_EXIT,
    EVENT_INVALIDATE_DENTRY, // deprecated
    EVENT_SETUID,
    EVENT_SETGID,
    EVENT_CAPSET,
    EVENT_ARGS_ENVS,
    EVENT_MOUNT_RELEASED,
    EVENT_SELINUX,
    EVENT_BPF,
    EVENT_PTRACE,
    EVENT_MMAP,
    EVENT_MPROTECT,
    EVENT_INIT_MODULE,
    EVENT_DELETE_MODULE,
    EVENT_SIGNAL,
    EVENT_SPLICE,
    EVENT_CGROUP_TRACING,
    EVENT_DNS,
    EVENT_NET_DEVICE,
    EVENT_VETH_PAIR,
    EVENT_BIND,
    EVENT_UNSHARE_MNTNS,
    EVENT_SYSCALLS,
    EVENT_ANOMALY_DETECTION_SYSCALL,
    EVENT_SYNTHETIC,
    EVENT_MAX, // has to be the last one

    EVENT_ALL = 0xffffffff // used as a mask for all the events
};

#define EVENT_LAST_APPROVER EVENT_SPLICE

enum {
    EVENT_FLAGS_ASYNC = 1<<0, // async, mostly io_uring
    EVENT_FLAGS_SAVED_BY_AD = 1<<1, // event send because of activity dump
    EVENT_FLAGS_ACTIVITY_DUMP_SAMPLE = 1<<2, // event is a AD sample
};

enum file_flags {
    LOWER_LAYER = 1 << 0,
    UPPER_LAYER = 1 << 1,
};

enum {
    SYNC_SYSCALL = 0,
    ASYNC_SYSCALL
};

enum {
    ACTIVITY_DUMP_RUNNING = 1<<0, // defines if an activity dump is running
    SAVED_BY_ACTIVITY_DUMP = 1<<1, // defines if the dentry should have been discarded, but was saved because of an activity dump
};

enum policy_mode {
    NO_FILTER = 0,
    ACCEPT = 1,
    DENY = 2,
};

enum policy_flags {
    BASENAME = 1,
    FLAGS = 2,
    MODE = 4,
    PARENT_NAME = 8,
};

enum tls_format {
   DEFAULT_TLS_FORMAT
};

typedef enum discard_check_state {
    NOT_DISCARDED,
    DISCARDED,
} discard_check_state;

enum bpf_cmd_def {
    BPF_MAP_CREATE_CMD,
    BPF_MAP_LOOKUP_ELEM_CMD,
    BPF_MAP_UPDATE_ELEM_CMD,
    BPF_MAP_DELETE_ELEM_CMD,
    BPF_MAP_GET_NEXT_KEY_CMD,
    BPF_PROG_LOAD_CMD,
    BPF_OBJ_PIN_CMD,
    BPF_OBJ_GET_CMD,
    BPF_PROG_ATTACH_CMD,
    BPF_PROG_DETACH_CMD,
    BPF_PROG_TEST_RUN_CMD,
    BPF_PROG_GET_NEXT_ID_CMD,
    BPF_MAP_GET_NEXT_ID_CMD,
    BPF_PROG_GET_FD_BY_ID_CMD,
    BPF_MAP_GET_FD_BY_ID_CMD,
    BPF_OBJ_GET_INFO_BY_FD_CMD,
    BPF_PROG_QUERY_CMD,
    BPF_RAW_TRACEPOINT_OPEN_CMD,
    BPF_BTF_LOAD_CMD,
    BPF_BTF_GET_FD_BY_ID_CMD,
    BPF_TASK_FD_QUERY_CMD,
    BPF_MAP_LOOKUP_AND_DELETE_ELEM_CMD,
    BPF_MAP_FREEZE_CMD,
    BPF_BTF_GET_NEXT_ID_CMD,
    BPF_MAP_LOOKUP_BATCH_CMD,
    BPF_MAP_LOOKUP_AND_DELETE_BATCH_CMD,
    BPF_MAP_UPDATE_BATCH_CMD,
    BPF_MAP_DELETE_BATCH_CMD,
    BPF_LINK_CREATE_CMD,
    BPF_LINK_UPDATE_CMD,
    BPF_LINK_GET_FD_BY_ID_CMD,
    BPF_LINK_GET_NEXT_ID_CMD,
    BPF_ENABLE_STATS_CMD,
    BPF_ITER_CREATE_CMD,
    BPF_LINK_DETACH_CMD,
    BPF_PROG_BIND_MAP_CMD,
};

enum dr_kprobe_progs {
    DR_OPEN_CALLBACK_KPROBE_KEY = 1,
    DR_SETATTR_CALLBACK_KPROBE_KEY,
    DR_MKDIR_CALLBACK_KPROBE_KEY,
    DR_MOUNT_STAGE_ONE_CALLBACK_KPROBE_KEY,
    DR_MOUNT_STAGE_TWO_CALLBACK_KPROBE_KEY,
    DR_SECURITY_INODE_RMDIR_CALLBACK_KPROBE_KEY,
    DR_SETXATTR_CALLBACK_KPROBE_KEY,
    DR_UNLINK_CALLBACK_KPROBE_KEY,
    DR_LINK_SRC_CALLBACK_KPROBE_KEY,
    DR_LINK_DST_CALLBACK_KPROBE_KEY,
    DR_RENAME_CALLBACK_KPROBE_KEY,
    DR_SELINUX_CALLBACK_KPROBE_KEY,
    DR_CHDIR_CALLBACK_KPROBE_KEY,
};

enum dr_tracepoint_progs {
    DR_OPEN_CALLBACK_TRACEPOINT_KEY = 1,
    DR_MKDIR_CALLBACK_TRACEPOINT_KEY,
    DR_MOUNT_STAGE_ONE_CALLBACK_TRACEPOINT_KEY,
    DR_MOUNT_STAGE_TWO_CALLBACK_TRACEPOINT_KEY,
    DR_LINK_DST_CALLBACK_TRACEPOINT_KEY,
    DR_RENAME_CALLBACK_TRACEPOINT_KEY,
    DR_CHDIR_CALLBACK_TRACEPOINT_KEY,
};

enum erpc_op {
    UNKNOWN_OP,
    DISCARD_INODE_OP,
    DISCARD_PID_OP,
    RESOLVE_SEGMENT_OP, // DEPRECATED
    RESOLVE_PATH_OP,
    RESOLVE_PARENT_OP, // DEPRECATED
    REGISTER_SPAN_TLS_OP, // can be used outside of the CWS, do not change the value
    EXPIRE_INODE_DISCARDER_OP,
    EXPIRE_PID_DISCARDER_OP,
    BUMP_DISCARDERS_REVISION,
    GET_RINGBUF_USAGE,
    USER_SESSION_CONTEXT_OP,
};

enum selinux_source_event_t {
    SELINUX_BOOL_CHANGE_SOURCE_EVENT,
    SELINUX_DISABLE_CHANGE_SOURCE_EVENT,
    SELINUX_ENFORCE_CHANGE_SOURCE_EVENT,
    SELINUX_BOOL_COMMIT_SOURCE_EVENT,
};

enum selinux_event_kind_t {
    SELINUX_BOOL_CHANGE_EVENT_KIND,
    SELINUX_STATUS_CHANGE_EVENT_KIND,
    SELINUX_BOOL_COMMIT_EVENT_KIND,
};

#endif
