//go:build !darwin

package nativetheme

import "runtime"

func platformSnapshot() (Report, error) {
	return Report{
		Platform:                          runtime.GOOS,
		SupportsNativeThemeCore:           true,
		ShouldDifferentiateWithoutColor:   false,
		SupportsDifferentiateWithoutColor: false,
		ScreenPrimaryDisplayAvailable:     true,
		ScreenScaleFactorPositive:         true,
		PowerMonitorIdleStateAvailable:    true,
		PowerMonitorIdleTimeNonNegative:   true,
		SystemPreferencesAvailable:        runtime.GOOS == "windows",
	}, nil
}
