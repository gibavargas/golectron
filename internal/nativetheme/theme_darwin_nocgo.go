//go:build darwin && !cgo

package nativetheme

func platformSnapshot() (Report, error) {
	return Report{
		Platform:                          "darwin",
		SupportsNativeThemeCore:           true,
		ShouldDifferentiateWithoutColor:   false,
		SupportsDifferentiateWithoutColor: false,
		ScreenPrimaryDisplayAvailable:     true,
		ScreenScaleFactorPositive:         true,
		PowerMonitorIdleStateAvailable:    true,
		PowerMonitorIdleTimeNonNegative:   true,
		SystemPreferencesAvailable:        true,
	}, nil
}
