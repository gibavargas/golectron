#include "cef_shim.h"

#if defined(ELECTRON_GO_HAS_CEF)
#include "include/cef_api_hash.h"
#include "include/capi/cef_app_capi.h"
#include "include/capi/cef_browser_capi.h"
#include "include/capi/cef_client_capi.h"
#include "include/capi/cef_life_span_handler_capi.h"
#include "include/capi/cef_load_handler_capi.h"
#include "include/internal/cef_string.h"
#include <limits.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

extern void goOnContextInitialized(void);
extern void goOnBrowserAfterCreated(int browser_id);
extern void goOnBrowserLoadEnd(int browser_id, int http_status_code);
extern void goOnBrowserLoadError(int browser_id, int error_code);
extern void goOnBrowserBeforeClose(int browser_id);

static cef_browser_process_handler_t* g_browser_process_handler = NULL;
static cef_life_span_handler_t* g_life_span_handler = NULL;
static cef_load_handler_t* g_load_handler = NULL;
static cef_client_t* g_client = NULL;
static cef_browser_t* g_browser = NULL;
static int g_cef_initialized = 0;
static int g_browser_id = 0;
static int g_load_complete = 0;
static int g_load_failed = 0;
static int g_close_requested = 0;
static int g_browser_closed = 0;

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

static void eg_cef_append_ascii_switch(
    struct _cef_command_line_t* command_line,
    const char* name) {
  if (!command_line || !command_line->append_switch || !name) {
    return;
  }
  cef_string_t switch_name;
  memset(&switch_name, 0, sizeof(switch_name));
  if (cef_string_from_ascii(name, strlen(name), &switch_name)) {
    command_line->append_switch(command_line, &switch_name);
    cef_string_clear(&switch_name);
  }
}

static int eg_cef_set_ascii_string(const char* source, cef_string_t* target) {
  if (!source || !target) {
    return 1;
  }
  return cef_string_from_ascii(source, strlen(source), target);
}

static void CEF_CALLBACK eg_cef_on_before_command_line_processing(
    struct _cef_app_t* self,
    const cef_string_t* process_type,
    struct _cef_command_line_t* command_line) {
  (void)self;
  (void)process_type;
  eg_cef_append_ascii_switch(command_line, "disable-background-networking");
  eg_cef_append_ascii_switch(command_line, "disable-breakpad");
  eg_cef_append_ascii_switch(command_line, "disable-component-update");
  eg_cef_append_ascii_switch(command_line, "disable-default-apps");
  eg_cef_append_ascii_switch(command_line, "disable-extensions");
  eg_cef_append_ascii_switch(command_line, "disable-gpu");
  eg_cef_append_ascii_switch(command_line, "disable-sync");
  eg_cef_append_ascii_switch(command_line, "metrics-recording-only");
  eg_cef_append_ascii_switch(command_line, "no-default-browser-check");
  eg_cef_append_ascii_switch(command_line, "no-first-run");
  eg_cef_append_ascii_switch(command_line, "no-zygote");
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

static void eg_cef_release_browser_ref(void) {
  if (!g_browser) {
    return;
  }
  g_browser->base.release((cef_base_ref_counted_t*)g_browser);
  g_browser = NULL;
}

static void eg_cef_store_browser_ref(struct _cef_browser_t* browser) {
  if (!browser || browser == g_browser) {
    return;
  }
  eg_cef_release_browser_ref();
  browser->base.add_ref((cef_base_ref_counted_t*)browser);
  g_browser = browser;
}

static void eg_cef_request_close_browser(struct _cef_browser_t* browser) {
  if (g_close_requested) {
    return;
  }
  if (!browser) {
    browser = g_browser;
  }
  if (!browser || !browser->get_host) {
    return;
  }
  cef_browser_host_t* host = browser->get_host(browser);
  if (!host || !host->close_browser) {
    return;
  }
  g_close_requested = 1;
  host->close_browser(host, 1);
}

static void CEF_CALLBACK eg_cef_on_before_close(
    struct _cef_life_span_handler_t* self,
    struct _cef_browser_t* browser);

static void CEF_CALLBACK eg_cef_on_after_created(
    struct _cef_life_span_handler_t* self,
    struct _cef_browser_t* browser) {
  (void)self;
  eg_cef_store_browser_ref(browser);
  if (browser && browser->get_identifier) {
    g_browser_id = browser->get_identifier(browser);
    goOnBrowserAfterCreated(g_browser_id);
  }
}

static cef_life_span_handler_t* eg_cef_make_life_span_handler(void) {
  cef_life_span_handler_t* handler =
      (cef_life_span_handler_t*)calloc(1, sizeof(cef_life_span_handler_t));
  if (!handler) {
    return NULL;
  }
  eg_cef_init_base(&handler->base, sizeof(cef_life_span_handler_t));
  handler->on_after_created = eg_cef_on_after_created;
  handler->on_before_close = eg_cef_on_before_close;
  return handler;
}

static int eg_cef_is_main_frame(struct _cef_frame_t* frame) {
  return frame && frame->is_main && frame->is_main(frame);
}

static int eg_cef_browser_id(struct _cef_browser_t* browser) {
  if (browser && browser->get_identifier) {
    return browser->get_identifier(browser);
  }
  return g_browser_id;
}

static void CEF_CALLBACK eg_cef_on_before_close(
    struct _cef_life_span_handler_t* self,
    struct _cef_browser_t* browser) {
  (void)self;
  g_browser_id = eg_cef_browser_id(browser);
  g_browser_closed = 1;
  goOnBrowserBeforeClose(g_browser_id);
  eg_cef_release_browser_ref();
  cef_quit_message_loop();
}

static void CEF_CALLBACK eg_cef_on_load_end(
    struct _cef_load_handler_t* self,
    struct _cef_browser_t* browser,
    struct _cef_frame_t* frame,
    int httpStatusCode) {
  (void)self;
  if (!eg_cef_is_main_frame(frame)) {
    return;
  }
  g_load_complete = 1;
  g_browser_id = eg_cef_browser_id(browser);
  goOnBrowserLoadEnd(g_browser_id, httpStatusCode);
  eg_cef_request_close_browser(browser);
}

static void CEF_CALLBACK eg_cef_on_load_error(
    struct _cef_load_handler_t* self,
    struct _cef_browser_t* browser,
    struct _cef_frame_t* frame,
    cef_errorcode_t errorCode,
    const cef_string_t* errorText,
    const cef_string_t* failedUrl) {
  (void)self;
  (void)errorText;
  (void)failedUrl;
  if (!eg_cef_is_main_frame(frame)) {
    return;
  }
  if ((int)errorCode == -3) {
    return;
  }
  g_load_failed = 1;
  g_browser_id = eg_cef_browser_id(browser);
  goOnBrowserLoadError(g_browser_id, (int)errorCode);
  eg_cef_request_close_browser(browser);
  cef_quit_message_loop();
}

static void CEF_CALLBACK eg_cef_on_loading_state_change(
    struct _cef_load_handler_t* self,
    struct _cef_browser_t* browser,
    int isLoading,
    int canGoBack,
    int canGoForward) {
  (void)self;
  (void)canGoBack;
  (void)canGoForward;
  if (isLoading || g_load_failed) {
    return;
  }
  g_load_complete = 1;
  g_browser_id = eg_cef_browser_id(browser);
  goOnBrowserLoadEnd(g_browser_id, 0);
  eg_cef_request_close_browser(browser);
}

static cef_load_handler_t* eg_cef_make_load_handler(void) {
  cef_load_handler_t* handler =
      (cef_load_handler_t*)calloc(1, sizeof(cef_load_handler_t));
  if (!handler) {
    return NULL;
  }
  eg_cef_init_base(&handler->base, sizeof(cef_load_handler_t));
  handler->on_loading_state_change = eg_cef_on_loading_state_change;
  handler->on_load_end = eg_cef_on_load_end;
  handler->on_load_error = eg_cef_on_load_error;
  return handler;
}

static cef_life_span_handler_t* CEF_CALLBACK
eg_cef_get_life_span_handler(struct _cef_client_t* self) {
  (void)self;
  return g_life_span_handler;
}

static cef_load_handler_t* CEF_CALLBACK
eg_cef_get_load_handler(struct _cef_client_t* self) {
  (void)self;
  return g_load_handler;
}

static cef_client_t* eg_cef_make_client(void) {
  cef_client_t* client = (cef_client_t*)calloc(1, sizeof(cef_client_t));
  if (!client) {
    return NULL;
  }
  eg_cef_init_base(&client->base, sizeof(cef_client_t));
  g_life_span_handler = eg_cef_make_life_span_handler();
  g_load_handler = eg_cef_make_load_handler();
  client->get_life_span_handler = eg_cef_get_life_span_handler;
  client->get_load_handler = eg_cef_get_load_handler;
  return client;
}

cef_app_t* eg_cef_make_app(void) {
  cef_app_t* app = (cef_app_t*)calloc(1, sizeof(cef_app_t));
  if (!app) {
    return NULL;
  }
  eg_cef_init_base(&app->base, sizeof(cef_app_t));
  g_browser_process_handler = eg_cef_make_browser_process_handler();
  app->on_before_command_line_processing =
      eg_cef_on_before_command_line_processing;
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
  cef_string_clear(&settings->log_file);
}

static void eg_cef_reset_browser_state(void) {
  g_browser_id = 0;
  g_load_complete = 0;
  g_load_failed = 0;
  g_close_requested = 0;
  g_browser_closed = 0;
  eg_cef_release_browser_ref();
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
  if (!eg_cef_set_cef_string(
          &request->settings.cache_path, &settings.root_cache_path) ||
      !eg_cef_set_cef_string(
          &request->settings.cache_path, &settings.cache_path) ||
      !eg_cef_set_ascii_string("/dev/null", &settings.log_file)) {
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
  if (!request || !out_result ||
      request->abi_revision != EG_BRIDGE_ABI_REVISION ||
      !request->url.data || request->url.len == 0 ||
      request->width <= 0 || request->height <= 0) {
    return EG_BRIDGE_STATUS_INVALID_REQUEST;
  }

  out_result->abi_revision = EG_BRIDGE_ABI_REVISION;
  out_result->status = EG_BRIDGE_STATUS_STARTING;
  out_result->browser_id = 0;
  eg_cef_reset_browser_state();

  if (!g_client) {
    g_client = eg_cef_make_client();
  }
  if (!g_client) {
    out_result->status = EG_BRIDGE_STATUS_FAILED;
    return EG_BRIDGE_STATUS_FAILED;
  }

  cef_window_info_t window_info;
  memset(&window_info, 0, sizeof(window_info));
  window_info.size = sizeof(window_info);
  window_info.bounds.x = 0;
  window_info.bounds.y = 0;
  window_info.bounds.width = request->width;
  window_info.bounds.height = request->height;
  window_info.runtime_style = CEF_RUNTIME_STYLE_ALLOY;

  cef_browser_settings_t browser_settings;
  memset(&browser_settings, 0, sizeof(browser_settings));
  browser_settings.size = sizeof(browser_settings);

  cef_string_t url;
  memset(&url, 0, sizeof(url));
  if (!eg_cef_set_cef_string(&request->url, &url)) {
    out_result->status = EG_BRIDGE_STATUS_INVALID_REQUEST;
    return EG_BRIDGE_STATUS_INVALID_REQUEST;
  }

  cef_browser_t* browser = cef_browser_host_create_browser_sync(
      &window_info, g_client, &url, &browser_settings, NULL, NULL);
  cef_string_clear(&url);
  cef_string_clear(&window_info.window_name);

  if (!browser) {
    out_result->status = EG_BRIDGE_STATUS_FAILED;
    return EG_BRIDGE_STATUS_FAILED;
  }
  if (browser->get_identifier) {
    g_browser_id = browser->get_identifier(browser);
    out_result->browser_id = g_browser_id;
  }

  out_result->status = EG_BRIDGE_STATUS_RUNNING;
  return EG_BRIDGE_STATUS_RUNNING;
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
  if (!g_cef_initialized) {
    return EG_BRIDGE_STATUS_INVALID_REQUEST;
  }
  cef_run_message_loop();
  if (g_load_failed || !g_load_complete || !g_browser_closed) {
    return EG_BRIDGE_STATUS_FAILED;
  }
  return EG_BRIDGE_STATUS_STOPPED;
}

eg_bridge_status eg_cef_shim_shutdown(eg_bridge_handle bridge) {
  (void)bridge;
  eg_cef_release_browser_ref();
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
