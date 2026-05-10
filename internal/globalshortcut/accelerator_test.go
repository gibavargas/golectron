package globalshortcut

import (
	"errors"
	"testing"
)

func TestParseAcceleratorCanonicalizesAliases(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "CmdOrCtrl+X", want: "CommandOrControl+X"},
		{in: " CommandOrControl + shift + p ", want: "CommandOrControl+Shift+P"},
		{in: "Ctrl+Option+Delete", want: "Control+Alt+Delete"},
		{in: "Meta+AltGr+Space", want: "Super+AltGr+Space"},
		{in: "a", want: "A"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			accelerator, err := ParseAccelerator(tt.in)
			if err != nil {
				t.Fatalf("ParseAccelerator() error = %v", err)
			}
			if got := accelerator.String(); got != tt.want {
				t.Fatalf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseAcceleratorRejectsMalformedInput(t *testing.T) {
	tests := []string{
		"",
		"CommandOrControl+",
		"CommandOrControl++X",
		"X+Shift",
		"Shift+Shift+X",
	}

	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			_, err := ParseAccelerator(tt)
			if err == nil {
				t.Fatal("ParseAccelerator() error = nil, want error")
			}
			if tt == "" && !errors.Is(err, ErrAcceleratorRequired) {
				t.Fatalf("ParseAccelerator() error = %v, want ErrAcceleratorRequired", err)
			}
			if tt != "" && !errors.Is(err, ErrInvalidAccelerator) {
				t.Fatalf("ParseAccelerator() error = %v, want ErrInvalidAccelerator", err)
			}
		})
	}
}
