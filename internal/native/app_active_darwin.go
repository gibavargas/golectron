//go:build darwin && cgo

package native

/*
#cgo LDFLAGS: -framework AppKit
int electron_go_app_is_active(void);
*/
import "C"

func platformAppActivity() (AppActivity, error) {
	return AppActivity{
		Platform:  "darwin",
		Supported: true,
		Active:    C.electron_go_app_is_active() == 1,
	}, nil
}
