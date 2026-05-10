//go:build darwin && !cgo

package nativetheme

func platformSnapshot() (Report, error) {
	return Report{
		Platform:                          "darwin",
		ShouldDifferentiateWithoutColor:   false,
		SupportsDifferentiateWithoutColor: false,
	}, nil
}
