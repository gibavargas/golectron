package webauthn

import (
	"errors"
	"testing"
)

func TestConfigureTouchID(t *testing.T) {
	var manager Manager
	if err := manager.Configure(Config{TouchID: &TouchIDConfig{KeychainAccessGroup: " TEAM.bundle "}}); err != nil {
		t.Fatalf("Configure() error = %v", err)
	}
	config := manager.Config()
	if config.TouchID == nil || config.TouchID.KeychainAccessGroup != "TEAM.bundle" {
		t.Fatalf("Config() = %#v", config)
	}
	config.TouchID.KeychainAccessGroup = "mutated"
	if got := manager.Config().TouchID.KeychainAccessGroup; got != "TEAM.bundle" {
		t.Fatalf("Config() returned mutable TouchID config: %q", got)
	}
}

func TestConfigureRejectsInvalidTouchID(t *testing.T) {
	var manager Manager
	cases := []Config{
		{TouchID: &TouchIDConfig{}},
		{TouchID: &TouchIDConfig{KeychainAccessGroup: "bad\x00group"}},
	}
	for _, tc := range cases {
		if err := manager.Configure(tc); !errors.Is(err, ErrInvalidConfig) {
			t.Fatalf("Configure(%#v) error = %v, want ErrInvalidConfig", tc, err)
		}
	}
}

func TestSupportsTouchID(t *testing.T) {
	if !SupportsTouchID("darwin") {
		t.Fatal("SupportsTouchID(darwin) = false, want true")
	}
	if SupportsTouchID("linux") {
		t.Fatal("SupportsTouchID(linux) = true, want false")
	}
}
