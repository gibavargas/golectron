//go:build linux && amd64 && cgo && electron_go_cef

package native

/*
#cgo CFLAGS: -DELECTRON_GO_HAS_CEF -I${SRCDIR}/../../native -I${SRCDIR}/../../native/cef/current
#cgo LDFLAGS: -L${SRCDIR}/../../bin -lcef -Wl,-rpath,$ORIGIN
#include "cef_shim.h"
*/
import "C"

import (
	"context"
	"sync/atomic"
)

var cefContextInitialized atomic.Bool

type CEFBridge struct{}

func NewBridge() Bridge {
	return CEFBridge{}
}

func (b CEFBridge) Start(ctx context.Context, req StartRequest) (*StartResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := ValidateStartRequest(req); err != nil {
		return nil, err
	}
	return nil, ErrBridgeUnavailable
}

//export goOnContextInitialized
func goOnContextInitialized() {
	cefContextInitialized.Store(true)
}
