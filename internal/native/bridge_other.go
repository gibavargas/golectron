//go:build !darwin && !linux && !windows

package native

func NewBridge() Bridge {
	return StubBridge{}
}
