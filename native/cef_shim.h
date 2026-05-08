#ifndef ELECTRON_GO_CEF_SHIM_H_
#define ELECTRON_GO_CEF_SHIM_H_

#include "electron_go_bridge.h"

#ifdef __cplusplus
extern "C" {
#endif

/*
 * This header is the C intermediary between CEF's callback/vtable structs and
 * Go. The real CEF implementation is compiled only when the native build sets
 * ELECTRON_GO_HAS_CEF and provides CEF headers/libraries.
 */
eg_bridge_status eg_cef_shim_execute_process(
    const eg_cef_execute_process_request* request,
    eg_cef_subprocess_result* out_result);

eg_bridge_status eg_cef_shim_initialize(
    eg_bridge_handle bridge,
    const eg_cef_initialize_request* request);

eg_bridge_status eg_cef_shim_create_browser_sync(
    eg_bridge_handle bridge,
    const eg_browser_window_create_request* request,
    eg_browser_window_result* out_result);

eg_bridge_status eg_cef_shim_load_url(
    eg_bridge_handle bridge,
    const eg_browser_window_load_request* request);

eg_bridge_status eg_cef_shim_run_message_loop(eg_bridge_handle bridge);
eg_bridge_status eg_cef_shim_shutdown(eg_bridge_handle bridge);

#ifdef __cplusplus
}
#endif

#endif
