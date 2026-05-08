#include "cef_shim.h"

#if defined(ELECTRON_GO_HAS_CEF)
#include "include/cef_api_hash.h"
#include "include/capi/cef_app_capi.h"
#include "include/internal/cef_string.h"
#include <limits.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

extern void goOnContextInitialized(void);

static cef_browser_process_handler_t* g_browser_process_handler = NULL;
static int g_cef_initialized = 0;

typedef struct eg_cef_argv_storage {
  int argc;
  char** argv;
} eg_cef_argv_storage;

static void CEF_CALLBACK eg_cef_base_add_ref(
    struct _cef_base_ref_counted_t* self) {
  (void)self;
}

static int CEF_CALLBACK eg_cef_base_release(
    struct _cef_base_ref_counted_t* self) {
  (void)self;
  return 0;
}

static int CEF_CALLBACK eg_cef_base_has_one_ref(
    struct _cef_base_ref_counted_t* self) {
  (void)self;
  return 1;
}

static int CEF_CALLBACK eg_cef_base_has_at_least_one_ref(
    struct _cef_base_ref_counted_t* self) {
  (void)self;
  return 1;
}

static void eg_cef_init_base(cef_base_ref_counted_t* base, size_t size) {
  base->size = size;
  base->add_ref = eg_cef_base_add_ref;
  base->release = eg_cef_base_release;
  base->has_one_ref = eg_cef_base_has_one_ref;
#if defined(CEF_API_VERSION) && CEF_API_VERSION >= 109
  base->has_at_least_one_ref = eg_cef_base_has_at_least_one_ref;
#endif
}

static void CEF_CALLBACK eg_cef_on_context_initialized(
    struct _cef_browser_process_handler_t* self) {
  (void)self;
  goOnContextInitialized();
}

static cef_browser_process_handler_t* eg_cef_make_browser_process_handler(void) {
  cef_browser_process_handler_t* handler =
      (cef_browser_process_handler_t*)calloc(1, sizeof(cef_browser_process_handler_t));
  if (!handler) {
    return NULL;
  }
  eg_cef_init_base(&handler->base, sizeof(cef_browser_process_handler_t));
  handler->on_context_initialized = eg_cef_on_context_initialized;
  return handler;
}

static cef_browser_process_handler_t* CEF_CALLBACK
eg_cef_get_browser_process_handler(struct _cef_app_t* self) {
  (void)self;
  return g_browser_process_handler;
}

cef_app_t* eg_cef_make_app(void) {
  cef_app_t* app = (cef_app_t*)calloc(1, sizeof(cef_app_t));
  if (!app) {
    return NULL;
  }
  eg_cef_init_base(&app->base, sizeof(cef_app_t));
  g_browser_process_handler = eg_cef_make_browser_process_handler();
  app->get_browser_process_handler = eg_cef_get_browser_process_handler;
  return app;
}

cef_app_t* make_cef_app(void) {
  return eg_cef_make_app();
}

const char* eg_cef_shim_link_proof(void) {
  return cef_api_hash(CEF_API_VERSION, 0);
}

static int eg_cef_configure_api_version(void) {
  return cef_api_hash(CEF_API_VERSION, 0) != NULL;
}

static void eg_cef_free_argv_storage(eg_cef_argv_storage* storage) {
  if (!storage || !storage->argv) {
    return;
  }
  for (int i = 0; i < storage->argc; i++) {
    free(storage->argv[i]);
  }
  free(storage->argv);
  storage->argv = NULL;
  storage->argc = 0;
}

static int eg_cef_copy_string_view(const eg_string_view* source, char** target) {
  if (!target) {
    return 0;
  }
  *target = NULL;
  if (!source || !source->data || source->len == 0) {
    *target = (char*)calloc(1, 1);
    return *target != NULL;
  }
  if (source->len > (uint64_t)SIZE_MAX - 1) {
    return 0;
  }
  char* copy = (char*)malloc((size_t)source->len + 1);
  if (!copy) {
    return 0;
  }
  memcpy(copy, source->data, (size_t)source->len);
  copy[(size_t)source->len] = '\0';
  *target = copy;
  return 1;
}

static int eg_cef_make_main_args(
    uint64_t argc,
    const eg_string_view* argv,
    cef_main_args_t* out_args,
    eg_cef_argv_storage* storage) {
  if (!out_args || !storage || argc > (uint64_t)INT_MAX) {
    return 0;
  }
  memset(out_args, 0, sizeof(*out_args));
  memset(storage, 0, sizeof(*storage));
  if (argc == 0) {
    return 1;
  }
  if (!argv) {
    return 0;
  }
  storage->argv = (char**)calloc((size_t)argc, sizeof(char*));
  if (!storage->argv) {
    return 0;
  }
  storage->argc = (int)argc;
  for (int i = 0; i < storage->argc; i++) {
    if (!eg_cef_copy_string_view(&argv[i], &storage->argv[i])) {
      eg_cef_free_argv_storage(storage);
      return 0;
    }
  }
  out_args->argc = storage->argc;
  out_args->argv = storage->argv;
  return 1;
}

static cef_log_severity_t eg_cef_to_log_severity(eg_cef_log_severity severity) {
  switch (severity) {
    case EG_CEF_LOG_SEVERITY_VERBOSE:
      return LOGSEVERITY_VERBOSE;
    case EG_CEF_LOG_SEVERITY_INFO:
      return LOGSEVERITY_INFO;
    case EG_CEF_LOG_SEVERITY_WARNING:
      return LOGSEVERITY_WARNING;
    case EG_CEF_LOG_SEVERITY_ERROR:
      return LOGSEVERITY_ERROR;
    case EG_CEF_LOG_SEVERITY_FATAL:
      return LOGSEVERITY_FATAL;
    case EG_CEF_LOG_SEVERITY_DISABLE:
      return LOGSEVERITY_DISABLE;
    case EG_CEF_LOG_SEVERITY_DEFAULT:
    default:
      return LOGSEVERITY_DEFAULT;
  }
}

static int eg_cef_set_cef_string(
    const eg_string_view* source,
    cef_string_t* target) {
  if (!source || !target || !source->data || source->len == 0) {
    return 1;
  }
  if (source->len > (uint64_t)SIZE_MAX) {
    return 0;
  }
  return cef_string_from_utf8(source->data, (size_t)source->len, target);
}

static void eg_cef_clear_settings(cef_settings_t* settings) {
  if (!settings) {
    return;
  }
  cef_string_clear(&settings->cache_path);
  cef_string_clear(&settings->root_cache_path);
}

eg_bridge_status eg_cef_shim_execute_process(
    const eg_cef_execute_process_request* request,
    eg_cef_subprocess_result* out_result) {
  if (!request || !out_result ||
      request->abi_revision != EG_BRIDGE_ABI_REVISION) {
    return EG_BRIDGE_STATUS_INVALID_REQUEST;
  }

  out_result->abi_revision = EG_BRIDGE_ABI_REVISION;
  out_result->exit_code = -1;
  out_result->status = EG_BRIDGE_STATUS_RUNNING;
  if (!eg_cef_configure_api_version()) {
    out_result->status = EG_BRIDGE_STATUS_FAILED;
    return EG_BRIDGE_STATUS_FAILED;
  }

  cef_main_args_t main_args;
  eg_cef_argv_storage storage;
  if (!eg_cef_make_main_args(
          request->argc, request->argv, &main_args, &storage)) {
    out_result->status = EG_BRIDGE_STATUS_INVALID_REQUEST;
    return EG_BRIDGE_STATUS_INVALID_REQUEST;
  }

  cef_app_t* app = eg_cef_make_app();
  if (!app) {
    eg_cef_free_argv_storage(&storage);
    out_result->status = EG_BRIDGE_STATUS_FAILED;
    return EG_BRIDGE_STATUS_FAILED;
  }

  int exit_code = cef_execute_process(&main_args, app, NULL);
  eg_cef_free_argv_storage(&storage);

  out_result->exit_code = exit_code;
  out_result->status =
      exit_code >= 0 ? EG_BRIDGE_STATUS_STOPPED : EG_BRIDGE_STATUS_RUNNING;
  return out_result->status;
}

eg_bridge_status eg_cef_shim_initialize(
    eg_bridge_handle bridge,
    const eg_cef_initialize_request* request) {
  (void)bridge;
  if (!request || request->abi_revision != EG_BRIDGE_ABI_REVISION) {
    return EG_BRIDGE_STATUS_INVALID_REQUEST;
  }
  if (g_cef_initialized) {
    return EG_BRIDGE_STATUS_RUNNING;
  }
  if (!eg_cef_configure_api_version()) {
    return EG_BRIDGE_STATUS_FAILED;
  }

  cef_main_args_t main_args;
  eg_cef_argv_storage storage;
  if (!eg_cef_make_main_args(
          request->argc, request->argv, &main_args, &storage)) {
    return EG_BRIDGE_STATUS_INVALID_REQUEST;
  }

  cef_settings_t settings;
  memset(&settings, 0, sizeof(settings));
  settings.size = sizeof(settings);
  settings.no_sandbox = request->settings.no_sandbox ? 1 : 0;
  settings.log_severity =
      eg_cef_to_log_severity(request->settings.log_severity);
  if (!eg_cef_set_cef_string(&request->app_dir, &settings.root_cache_path) ||
      !eg_cef_set_cef_string(
          &request->settings.cache_path, &settings.cache_path)) {
    eg_cef_free_argv_storage(&storage);
    eg_cef_clear_settings(&settings);
    return EG_BRIDGE_STATUS_INVALID_REQUEST;
  }

  cef_app_t* app = eg_cef_make_app();
  if (!app) {
    eg_cef_free_argv_storage(&storage);
    eg_cef_clear_settings(&settings);
    return EG_BRIDGE_STATUS_FAILED;
  }

  int ok = cef_initialize(&main_args, &settings, app, NULL);
  eg_cef_free_argv_storage(&storage);
  eg_cef_clear_settings(&settings);
  if (ok != 1) {
    return EG_BRIDGE_STATUS_FAILED;
  }
  g_cef_initialized = 1;
  return EG_BRIDGE_STATUS_RUNNING;
}

eg_bridge_status eg_cef_shim_create_browser_sync(
    eg_bridge_handle bridge,
    const eg_browser_window_create_request* request,
    eg_browser_window_result* out_result) {
  (void)bridge;
  (void)request;
  if (out_result) {
    out_result->abi_revision = EG_BRIDGE_ABI_REVISION;
    out_result->status = EG_BRIDGE_STATUS_UNAVAILABLE;
  }
  return EG_BRIDGE_STATUS_UNAVAILABLE;
}

eg_bridge_status eg_cef_shim_load_url(
    eg_bridge_handle bridge,
    const eg_browser_window_load_request* request) {
  (void)bridge;
  (void)request;
  return EG_BRIDGE_STATUS_UNAVAILABLE;
}

eg_bridge_status eg_cef_shim_run_message_loop(eg_bridge_handle bridge) {
  (void)bridge;
  return EG_BRIDGE_STATUS_UNAVAILABLE;
}

eg_bridge_status eg_cef_shim_shutdown(eg_bridge_handle bridge) {
  (void)bridge;
  if (g_cef_initialized) {
    cef_shutdown();
    g_cef_initialized = 0;
  }
  return EG_BRIDGE_STATUS_STOPPED;
}

#else

eg_bridge_status eg_cef_shim_execute_process(
    const eg_cef_execute_process_request* request,
    eg_cef_subprocess_result* out_result) {
  (void)request;
  if (out_result) {
    out_result->abi_revision = EG_BRIDGE_ABI_REVISION;
    out_result->exit_code = 78;
    out_result->status = EG_BRIDGE_STATUS_UNAVAILABLE;
  }
  return EG_BRIDGE_STATUS_UNAVAILABLE;
}

eg_bridge_status eg_cef_shim_initialize(
    eg_bridge_handle bridge,
    const eg_cef_initialize_request* request) {
  (void)bridge;
  (void)request;
  return EG_BRIDGE_STATUS_UNAVAILABLE;
}

eg_bridge_status eg_cef_shim_create_browser_sync(
    eg_bridge_handle bridge,
    const eg_browser_window_create_request* request,
    eg_browser_window_result* out_result) {
  (void)bridge;
  (void)request;
  if (out_result) {
    out_result->abi_revision = EG_BRIDGE_ABI_REVISION;
    out_result->status = EG_BRIDGE_STATUS_UNAVAILABLE;
  }
  return EG_BRIDGE_STATUS_UNAVAILABLE;
}

eg_bridge_status eg_cef_shim_load_url(
    eg_bridge_handle bridge,
    const eg_browser_window_load_request* request) {
  (void)bridge;
  (void)request;
  return EG_BRIDGE_STATUS_UNAVAILABLE;
}

eg_bridge_status eg_cef_shim_run_message_loop(eg_bridge_handle bridge) {
  (void)bridge;
  return EG_BRIDGE_STATUS_UNAVAILABLE;
}

eg_bridge_status eg_cef_shim_shutdown(eg_bridge_handle bridge) {
  (void)bridge;
  return EG_BRIDGE_STATUS_UNAVAILABLE;
}

#endif
