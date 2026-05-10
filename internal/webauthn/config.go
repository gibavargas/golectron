package webauthn

import (
	"errors"
	"fmt"
	"strings"
)

var ErrInvalidConfig = errors.New("invalid WebAuthn configuration")

type TouchIDConfig struct {
	KeychainAccessGroup string
}

type Config struct {
	TouchID *TouchIDConfig
}

type Manager struct {
	config Config
}

func (m *Manager) Configure(config Config) error {
	normalized, err := NormalizeConfig(config)
	if err != nil {
		return err
	}
	m.config = normalized
	return nil
}

func (m *Manager) Config() Config {
	config := m.config
	if config.TouchID != nil {
		touchID := *config.TouchID
		config.TouchID = &touchID
	}
	return config
}

func NormalizeConfig(config Config) (Config, error) {
	if config.TouchID != nil {
		group := strings.TrimSpace(config.TouchID.KeychainAccessGroup)
		if group == "" || strings.Contains(group, "\x00") {
			return Config{}, fmt.Errorf("%w: touchID.keychainAccessGroup", ErrInvalidConfig)
		}
		config.TouchID = &TouchIDConfig{KeychainAccessGroup: group}
	}
	return config, nil
}

func SupportsTouchID(targetOS string) bool {
	return targetOS == "darwin" || targetOS == "macos"
}
