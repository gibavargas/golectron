package safestorage

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrUnavailable       = errors.New("safeStorage backend unavailable")
	ErrInvalidBackend    = errors.New("invalid safeStorage backend")
	ErrInvalidCiphertext = errors.New("invalid safeStorage ciphertext")
)

type Backend string

const (
	BackendBasicText Backend = "basic_text"
	BackendKeychain  Backend = "keychain"
	BackendLibsecret Backend = "libsecret"
	BackendKWallet   Backend = "kwallet"
)

type Storage struct {
	backend   Backend
	available bool
	key       []byte
}

func New(backend Backend, available bool, key []byte) (*Storage, error) {
	if backend == "" {
		backend = BackendBasicText
	}
	if !isValidBackend(backend) {
		return nil, fmt.Errorf("%w: %s", ErrInvalidBackend, backend)
	}
	if len(key) == 0 {
		key = []byte("electron-go-safe-storage")
	}
	return &Storage{
		backend:   backend,
		available: available,
		key:       append([]byte(nil), key...),
	}, nil
}

func (s *Storage) Backend() Backend {
	return s.backend
}

func (s *Storage) IsEncryptionAvailable() bool {
	return s.available
}

func (s *Storage) IsEncryptionAvailableAsync(ctx context.Context) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	return s.available, nil
}

func (s *Storage) EncryptString(plain string) ([]byte, error) {
	if !s.available {
		return nil, ErrUnavailable
	}
	mac := hmac.New(sha256.New, s.key)
	_, _ = mac.Write([]byte(s.backend))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write([]byte(plain))
	sum := mac.Sum(nil)
	payload := append([]byte(nil), sum...)
	payload = append(payload, []byte(plain)...)
	return []byte("egsafe:" + string(s.backend) + ":" + base64.StdEncoding.EncodeToString(payload)), nil
}

func (s *Storage) DecryptString(ciphertext []byte) (string, error) {
	if !s.available {
		return "", ErrUnavailable
	}
	parts := strings.SplitN(string(ciphertext), ":", 3)
	if len(parts) != 3 || parts[0] != "egsafe" || Backend(parts[1]) != s.backend {
		return "", ErrInvalidCiphertext
	}
	payload, err := base64.StdEncoding.DecodeString(parts[2])
	if err != nil || len(payload) < sha256.Size {
		return "", ErrInvalidCiphertext
	}
	sum := payload[:sha256.Size]
	plain := payload[sha256.Size:]
	mac := hmac.New(sha256.New, s.key)
	_, _ = mac.Write([]byte(s.backend))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write(plain)
	if !hmac.Equal(sum, mac.Sum(nil)) {
		return "", ErrInvalidCiphertext
	}
	return string(plain), nil
}

func (s *Storage) EncryptStringAsync(ctx context.Context, plain string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.EncryptString(plain)
}

func (s *Storage) DecryptStringAsync(ctx context.Context, ciphertext []byte) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return s.DecryptString(ciphertext)
}

func isValidBackend(backend Backend) bool {
	switch backend {
	case BackendBasicText, BackendKeychain, BackendLibsecret, BackendKWallet:
		return true
	default:
		return false
	}
}
