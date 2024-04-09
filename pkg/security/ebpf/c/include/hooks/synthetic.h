#ifndef _HOOKS_SYNTHETIC_H_
#define _HOOKS_SYNTHETIC_H_

#define PATH_SIZE 64

enum param_kind_t {
	PARAM_NO_ACTION,
	PARAM_KIND_INTEGER,
	PARAM_KIND_NULL_STR,
};

#define param_parsing(idx) \
	u64 param##idx##kind; \
    LOAD_CONSTANT("param" #idx "kind", param##idx##kind); \
                                             \
	switch (param##idx##kind) { \
	case PARAM_KIND_INTEGER: \
		value = CTX_PARM##idx(ctx); \
		bpf_printk("synthetic_hook, param%d `%d`", idx, value); \
		break; \
	case PARAM_KIND_NULL_STR: \
		path = (char *)CTX_PARM##idx(ctx); \
		bpf_probe_read_str(&buf, PATH_SIZE, path); \
		bpf_printk("synthetic_hook, param%d `%s`", idx, &buf); \
		break; \
	}

HOOK_ENTRY("synthetic_hook")
int hook_synthetic(ctx_t *ctx) {
	char buf[PATH_SIZE];
	char *path;
	u64 value;


	param_parsing(1);
	param_parsing(2);

    return 0;
}

#endif
