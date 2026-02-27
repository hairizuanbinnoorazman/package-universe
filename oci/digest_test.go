package oci

import (
	"io"
	"strings"
	"testing"
)

func TestParseDigest(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantAlgo  string
		wantHex   string
		wantError bool
	}{
		{
			name:     "valid sha256",
			input:    "sha256:abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			wantAlgo: "sha256",
			wantHex:  "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		},
		{
			name:      "empty string",
			input:     "",
			wantError: true,
		},
		{
			name:      "no colon",
			input:     "sha256abcdef",
			wantError: true,
		},
		{
			name:      "missing algorithm",
			input:     ":abcdef",
			wantError: true,
		},
		{
			name:      "uppercase hex",
			input:     "sha256:ABCDEF",
			wantError: true,
		},
		{
			name:      "invalid characters in hex",
			input:     "sha256:xyz123",
			wantError: true,
		},
		{
			name:     "short hex",
			input:    "sha256:abcd",
			wantAlgo: "sha256",
			wantHex:  "abcd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := ParseDigest(tt.input)
			if tt.wantError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if d.Algorithm != tt.wantAlgo {
				t.Errorf("algorithm = %q, want %q", d.Algorithm, tt.wantAlgo)
			}
			if d.Hex != tt.wantHex {
				t.Errorf("hex = %q, want %q", d.Hex, tt.wantHex)
			}
		})
	}
}

func TestDigestInfo_String(t *testing.T) {
	d := DigestInfo{Algorithm: "sha256", Hex: "abcdef"}
	got := d.String()
	want := "sha256:abcdef"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestDigestInfo_ShortHex(t *testing.T) {
	tests := []struct {
		hex  string
		want string
	}{
		{"abcdef", "ab"},
		{"a", "a"},
		{"", ""},
	}
	for _, tt := range tests {
		d := DigestInfo{Hex: tt.hex}
		got := d.ShortHex()
		if got != tt.want {
			t.Errorf("ShortHex(%q) = %q, want %q", tt.hex, got, tt.want)
		}
	}
}

func TestVerifyingReader(t *testing.T) {
	data := "hello world"
	vr := NewVerifyingReader(strings.NewReader(data))

	_, err := io.ReadAll(vr)
	if err != nil {
		t.Fatalf("failed to read: %v", err)
	}

	digest := vr.Digest()
	if digest.Algorithm != "sha256" {
		t.Errorf("algorithm = %q, want sha256", digest.Algorithm)
	}

	// SHA256 of "hello world"
	expectedHex := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	if digest.Hex != expectedHex {
		t.Errorf("hex = %q, want %q", digest.Hex, expectedHex)
	}

	if vr.Size() != int64(len(data)) {
		t.Errorf("size = %d, want %d", vr.Size(), len(data))
	}
}

func TestVerifyingReader_Verify(t *testing.T) {
	data := "hello world"
	vr := NewVerifyingReader(strings.NewReader(data))
	io.ReadAll(vr)

	t.Run("correct digest", func(t *testing.T) {
		expected := DigestInfo{
			Algorithm: "sha256",
			Hex:       "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
		}
		if err := vr.Verify(expected); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("wrong digest", func(t *testing.T) {
		expected := DigestInfo{
			Algorithm: "sha256",
			Hex:       "0000000000000000000000000000000000000000000000000000000000000000",
		}
		err := vr.Verify(expected)
		if err == nil {
			t.Error("expected error but got none")
		}
	})
}
