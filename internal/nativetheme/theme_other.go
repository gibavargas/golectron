//go:build !darwin

package nativetheme

import "runtime"

func platformSnapshot() (Report, error) {
	return Report{
		Platform:                          runtime.GOOS,
		ShouldDifferentiateWithoutColor:   false,
		SupportsDifferentiateWithoutColor: false,
	}, nil
}
