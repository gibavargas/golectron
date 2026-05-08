//go:build darwin

package native

func NewBridge() Bridge {
	return StubBridge{Platform: "darwin"}
}
