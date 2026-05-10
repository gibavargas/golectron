package safestorage

import (
	"context"
	"errors"
	"testing"
)

func TestStorageRoundTrip(t *testing.T) {
	storage, err := New(BackendKeychain, true, []byte("secret"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if storage.Backend() != BackendKeychain || !storage.IsEncryptionAvailable() {
		t.Fatalf("storage = backend %q available %v", storage.Backend(), storage.IsEncryptionAvailable())
	}
	ciphertext, err := storage.EncryptString("token")
	if err != nil {
		t.Fatalf("EncryptString() error = %v", err)
	}
	plain, err := storage.DecryptString(ciphertext)
	if err != nil {
		t.Fatalf("DecryptString() error = %v", err)
	}
	if plain != "token" {
		t.Fatalf("plain = %q, want token", plain)
	}
}

func TestStorageAsyncRoundTripAndAvailability(t *testing.T) {
	storage, err := New(BackendLibsecret, true, []byte("secret"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	available, err := storage.IsEncryptionAvailableAsync(context.Background())
	if err != nil {
		t.Fatalf("IsEncryptionAvailableAsync() error = %v", err)
	}
	if !available {
		t.Fatal("available = false, want true")
	}
	ciphertext, err := storage.EncryptStringAsync(context.Background(), "token")
	if err != nil {
		t.Fatalf("EncryptStringAsync() error = %v", err)
	}
	plain, err := storage.DecryptStringAsync(context.Background(), ciphertext)
	if err != nil {
		t.Fatalf("DecryptStringAsync() error = %v", err)
	}
	if plain != "token" {
		t.Fatalf("plain = %q, want token", plain)
	}
}

func TestStorageUnavailable(t *testing.T) {
	storage, err := New(BackendBasicText, false, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if storage.IsEncryptionAvailable() {
		t.Fatal("IsEncryptionAvailable() = true, want false")
	}
	if _, err := storage.EncryptString("token"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("EncryptString(unavailable) error = %v, want ErrUnavailable", err)
	}
	if _, err := storage.DecryptString([]byte("egsafe:basic_text:payload")); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("DecryptString(unavailable) error = %v, want ErrUnavailable", err)
	}
}

func TestStorageRejectsInvalidBackendAndCiphertext(t *testing.T) {
	if _, err := New("unknown", true, nil); !errors.Is(err, ErrInvalidBackend) {
		t.Fatalf("New(invalid backend) error = %v, want ErrInvalidBackend", err)
	}
	storage, err := New(BackendKWallet, true, []byte("secret"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if _, err := storage.DecryptString([]byte("bad")); !errors.Is(err, ErrInvalidCiphertext) {
		t.Fatalf("DecryptString(bad) error = %v, want ErrInvalidCiphertext", err)
	}
	ciphertext, err := storage.EncryptString("token")
	if err != nil {
		t.Fatalf("EncryptString() error = %v", err)
	}
	ciphertext[len(ciphertext)-1] = 'A'
	if _, err := storage.DecryptString(ciphertext); !errors.Is(err, ErrInvalidCiphertext) {
		t.Fatalf("DecryptString(tampered) error = %v, want ErrInvalidCiphertext", err)
	}
}

func TestStorageAsyncHonorsContextCancellation(t *testing.T) {
	storage, err := New(BackendBasicText, true, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := storage.IsEncryptionAvailableAsync(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("IsEncryptionAvailableAsync(canceled) error = %v, want context.Canceled", err)
	}
	if _, err := storage.EncryptStringAsync(ctx, "token"); !errors.Is(err, context.Canceled) {
		t.Fatalf("EncryptStringAsync(canceled) error = %v, want context.Canceled", err)
	}
	if _, err := storage.DecryptStringAsync(ctx, []byte("ciphertext")); !errors.Is(err, context.Canceled) {
		t.Fatalf("DecryptStringAsync(canceled) error = %v, want context.Canceled", err)
	}
}
