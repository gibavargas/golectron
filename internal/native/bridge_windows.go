//go:build windows

package native

func NewBridge() Bridge {
	return StubBridge{Platform: "windows"}
}
