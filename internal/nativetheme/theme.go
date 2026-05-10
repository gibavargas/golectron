package nativetheme

import "context"

type Report struct {
	Platform                          string `json:"platform"`
	SupportsNativeThemeCore           bool   `json:"supportsNativeThemeCore"`
	ShouldDifferentiateWithoutColor   bool   `json:"shouldDifferentiateWithoutColor"`
	SupportsDifferentiateWithoutColor bool   `json:"supportsDifferentiateWithoutColor"`
	ScreenPrimaryDisplayAvailable     bool   `json:"screenPrimaryDisplayAvailable"`
	ScreenScaleFactorPositive         bool   `json:"screenScaleFactorPositive"`
	PowerMonitorIdleStateAvailable    bool   `json:"powerMonitorIdleStateAvailable"`
	PowerMonitorIdleTimeNonNegative   bool   `json:"powerMonitorIdleTimeNonNegative"`
	SystemPreferencesAvailable        bool   `json:"systemPreferencesAvailable"`
}

func Snapshot(ctx context.Context) (Report, error) {
	if err := ctx.Err(); err != nil {
		return Report{}, err
	}
	return platformSnapshot()
}
