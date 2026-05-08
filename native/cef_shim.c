#include "cef_shim.h"

#if defined(ELECTRON_GO_HAS_CEF)
#include "include/capi/cef_app_capi.h"
#include "include/capi/cef_browser_capi.h"
#include "include/capi/cef_client_capi.h"

/*
 * TODO(cef_bootstrap): Populate cef_app_t and cef_client_t vtables here.
 * Go must not store Go function pointers in CEF structs directly. CEF calls C
 * functions from this file, and those C functions may call exported Go
 * callbacks after copying data out of CEF-owned memory.
 */

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
