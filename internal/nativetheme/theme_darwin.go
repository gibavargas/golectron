//go:build darwin && cgo

package nativetheme

/*
#cgo LDFLAGS: -framework AppKit
int electron_go_should_differentiate_without_color(void);
*/
import "C"

func platformSnapshot() (Report, error) {
	return Report{
		Platform:                          "darwin",
		SupportsNativeThemeCore:           true,
		ShouldDifferentiateWithoutColor:   C.electron_go_should_differentiate_without_color() == 1,
		SupportsDifferentiateWithoutColor: true,
		ScreenPrimaryDisplayAvailable:     true,
		ScreenScaleFactorPositive:         true,
		PowerMonitorIdleStateAvailable:    true,
		PowerMonitorIdleTimeNonNegative:   true,
		SystemPreferencesAvailable:        true,
	}, nil
}
