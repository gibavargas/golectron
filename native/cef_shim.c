#include "cef_shim.h"

#if defined(ELECTRON_GO_HAS_CEF)
#include "include/cef_api_hash.h"
#include "include/capi/cef_browser_capi.h"
#include "include/capi/cef_client_capi.h"
#include <stdlib.h>

extern void goOnContextInitialized(void);

static cef_browser_process_handler_t* g_browser_process_handler = NULL;

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
