//go:build linux && !electron_go_cef

package native

func NewBridge() Bridge {
	return StubBridge{Platform: "linux"}
}
