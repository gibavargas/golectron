//go:build !darwin

package native

func platformAppActivity() (AppActivity, error) {
	return unsupportedAppActivity()
}
