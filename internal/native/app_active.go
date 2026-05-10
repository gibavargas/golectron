package native

import (
	"context"
	"runtime"
)

// AppActivity reports Electron's app.isActive() foreground-state surface.
// Electron 42 exposes this API on macOS only.
type AppActivity struct {
	Platform  string `json:"platform"`
	Supported bool   `json:"supported"`
	Active    bool   `json:"active"`
}

func IsAppActive(ctx context.Context) (AppActivity, error) {
	if err := ctx.Err(); err != nil {
		return AppActivity{}, err
	}
	return platformAppActivity()
}

func unsupportedAppActivity() (AppActivity, error) {
	return AppActivity{
		Platform:  runtime.GOOS,
		Supported: false,
		Active:    false,
	}, nil
}
