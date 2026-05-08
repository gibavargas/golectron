//go:build linux

package native

func NewBridge() Bridge {
	return StubBridge{Platform: "linux"}
}
