//go:build darwin && !cgo

package native

func platformAppActivity() (AppActivity, error) {
	return AppActivity{
		Platform:  "darwin",
		Supported: false,
		Active:    false,
	}, nil
}
