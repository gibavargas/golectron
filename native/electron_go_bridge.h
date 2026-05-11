#ifndef ELECTRON_GO_BRIDGE_H_
#define ELECTRON_GO_BRIDGE_H_

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

#define EG_BRIDGE_ABI_REVISION 1u

typedef enum eg_bridge_status {
  EG_BRIDGE_STATUS_UNAVAILABLE = 0,
  EG_BRIDGE_STATUS_INVALID_REQUEST = 1,
  EG_BRIDGE_STATUS_STARTING = 2,
  EG_BRIDGE_STATUS_RUNNING = 3,
  EG_BRIDGE_STATUS_STOPPED = 4,
  EG_BRIDGE_STATUS_FAILED = 5
} eg_bridge_status;

typedef enum eg_cef_log_severity {
  EG_CEF_LOG_SEVERITY_DEFAULT = 0,
  EG_CEF_LOG_SEVERITY_VERBOSE = 1,
  EG_CEF_LOG_SEVERITY_INFO = 2,
  EG_CEF_LOG_SEVERITY_WARNING = 3,
  EG_CEF_LOG_SEVERITY_ERROR = 4,
  EG_CEF_LOG_SEVERITY_FATAL = 5,
  EG_CEF_LOG_SEVERITY_DISABLE = 6
} eg_cef_log_severity;

typedef struct eg_string_view {
  const char* data;
  uint64_t len;
} eg_string_view;

typedef struct eg_cef_settings {
  uint8_t no_sandbox;
  eg_string_view cache_path;
  eg_cef_log_severity log_severity;
} eg_cef_settings;

typedef struct eg_cef_execute_process_request {
  uint32_t abi_revision;
  uint64_t argc;
  const eg_string_view* argv;
} eg_cef_execute_process_request;

typedef struct eg_cef_subprocess_result {
  uint32_t abi_revision;
  int32_t exit_code;
  eg_bridge_status status;
  eg_string_view error_message;
} eg_cef_subprocess_result;

typedef struct eg_cef_initialize_request {
  uint32_t abi_revision;
  eg_string_view app_dir;
  uint64_t argc;
  const eg_string_view* argv;
  eg_cef_settings settings;
} eg_cef_initialize_request;

typedef struct eg_browser_window_create_request {
  uint32_t abi_revision;
  eg_string_view url;
  int32_t width;
  int32_t height;
  uint8_t show;
  uint8_t auto_close_on_load;
} eg_browser_window_create_request;

typedef struct eg_browser_window_load_request {
  uint32_t abi_revision;
  int64_t browser_id;
  eg_string_view url;
} eg_browser_window_load_request;

typedef struct eg_browser_window_result {
  uint32_t abi_revision;
  eg_bridge_status status;
  int64_t browser_id;
  eg_string_view error_message;
} eg_browser_window_result;

typedef struct eg_bridge_start_request {
  uint32_t abi_revision;
  eg_string_view app_dir;
  eg_string_view main_path;
  eg_string_view app_name;
  eg_string_view app_version;
  eg_string_view electron_version;
  uint64_t argc;
  const eg_string_view* argv;
  uint64_t envc;
  const eg_string_view* envp;
} eg_bridge_start_request;

typedef struct eg_engine_versions {
  eg_string_view chromium;
  eg_string_view node;
  eg_string_view v8;
} eg_engine_versions;

typedef struct eg_bridge_start_result {
  uint32_t abi_revision;
  eg_bridge_status status;
  int64_t pid;
  uint32_t window_count;
  eg_engine_versions engines;
  eg_string_view compatibility;
  eg_string_view bridge_revision;
  eg_string_view error_message;
} eg_bridge_start_result;

typedef void* eg_bridge_handle;

/* All lifecycle functions below must be called on the process main thread. */
eg_bridge_status eg_bridge_create(eg_bridge_handle* out_bridge);
eg_bridge_status eg_bridge_destroy(eg_bridge_handle bridge);
eg_bridge_status eg_bridge_start(
    eg_bridge_handle bridge,
    const eg_bridge_start_request* request,
    eg_bridge_start_result* out_result);
eg_bridge_status eg_bridge_free_result(eg_bridge_start_result* result);

eg_bridge_status eg_cef_execute_process(
    const eg_cef_execute_process_request* request,
    eg_cef_subprocess_result* out_result);
eg_bridge_status eg_cef_initialize(
    eg_bridge_handle bridge,
    const eg_cef_initialize_request* request);
eg_bridge_status eg_cef_create_browser_sync(
    eg_bridge_handle bridge,
    const eg_browser_window_create_request* request,
    eg_browser_window_result* out_result);
eg_bridge_status eg_cef_load_url(
    eg_bridge_handle bridge,
    const eg_browser_window_load_request* request);
eg_bridge_status eg_cef_run_message_loop(eg_bridge_handle bridge);
eg_bridge_status eg_cef_shutdown(eg_bridge_handle bridge);

#ifdef __cplusplus
}
#endif

#endif
