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

typedef struct eg_string_view {
  const char* data;
  uint64_t len;
} eg_string_view;

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

eg_bridge_status eg_bridge_create(eg_bridge_handle* out_bridge);
eg_bridge_status eg_bridge_destroy(eg_bridge_handle bridge);
eg_bridge_status eg_bridge_start(
    eg_bridge_handle bridge,
    const eg_bridge_start_request* request,
    eg_bridge_start_result* out_result);
eg_bridge_status eg_bridge_free_result(eg_bridge_start_result* result);

#ifdef __cplusplus
}
#endif

#endif
