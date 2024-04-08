#ifndef _HOOKS_SYNTHETIC_H_
#define _HOOKS_SYNTHETIC_H_

HOOK_ENTRY("synthetic_hook")
int hook_synthetic(ctx_t *ctx) {
    bpf_printk("synthetic_hook");
    return 0;
}

#endif
