#ifndef _HOOKS_SYNTHETIC_H_
#define _HOOKS_SYNTHETIC_H_

#define PER_ARG_SIZE 64

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
		bpf_probe_read(&event.data[(idx - 1) * PER_ARG_SIZE], sizeof(value), &value); \
		break; \
	case PARAM_KIND_NULL_STR: \
		buf = &event.data[(idx - 1) * PER_ARG_SIZE]; \
		path = (char *)CTX_PARM##idx(ctx); \
		bpf_probe_read_str(buf, PER_ARG_SIZE, path); \
		break; \
	}

HOOK_ENTRY("synthetic_hook")
int hook_synthetic(ctx_t *ctx) {
	u64 synth_id;
    LOAD_CONSTANT("synth_id", synth_id);

	struct synthetic_event_t event = {
		.synth_id = synth_id,
	};

	struct proc_cache_t *entry = fill_process_context(&event.process);
    fill_container_context(entry, &event.container);
    fill_span_context(&event.span);

	char *path;
	char *buf;
	u64 value;

	param_parsing(1);
	param_parsing(2);

	send_event(ctx, EVENT_SYNTHETIC, event);

    return 0;
}

#endif
