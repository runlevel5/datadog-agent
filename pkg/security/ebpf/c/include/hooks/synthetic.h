#ifndef _HOOKS_SYNTHETIC_H_
#define _HOOKS_SYNTHETIC_H_

HOOK_ENTRY("synthetic_hook")
int hook_synthetic(ctx_t *ctx) {
    char *path = (char *)CTX_PARM2(ctx);
    char buf[64];
    bpf_probe_read_str(&buf, 32, path);
    bpf_printk("synthetic_hook, `%s`", &buf);
    return 0;
}

#endif
