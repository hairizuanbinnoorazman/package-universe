package oci

import (
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"regexp"
	"strings"
)

var digestRegexp = regexp.MustCompile(`^([a-z0-9]+):([a-f0-9]+)$`)

// DigestInfo holds a parsed digest in algorithm:hex format.
type DigestInfo struct {
	Algorithm string
	Hex       string
}

// String returns the digest in algorithm:hex format.
func (d DigestInfo) String() string {
	return d.Algorithm + ":" + d.Hex
}

// ShortHex returns the first 2 characters of the hex string (used for sharded paths).
func (d DigestInfo) ShortHex() string {
	if len(d.Hex) < 2 {
		return d.Hex
	}
	return d.Hex[:2]
}

// ParseDigest parses a digest string in algorithm:hex format.
func ParseDigest(s string) (DigestInfo, error) {
	s = strings.TrimSpace(s)
	matches := digestRegexp.FindStringSubmatch(s)
	if matches == nil {
		return DigestInfo{}, fmt.Errorf("%w: %q", ErrInvalidDigest, s)
	}
	return DigestInfo{
		Algorithm: matches[1],
		Hex:       matches[2],
	}, nil
}

// VerifyingReader wraps a reader and computes sha256 as data is read.
type VerifyingReader struct {
	reader io.Reader
	hash   hash.Hash
	size   int64
}

// NewVerifyingReader creates a new VerifyingReader that computes sha256.
func NewVerifyingReader(r io.Reader) *VerifyingReader {
	h := sha256.New()
	return &VerifyingReader{
		reader: io.TeeReader(r, h),
		hash:   h,
	}
}

// Read implements io.Reader.
func (vr *VerifyingReader) Read(p []byte) (int, error) {
	n, err := vr.reader.Read(p)
	vr.size += int64(n)
	return n, err
}

// Digest returns the computed sha256 digest after all data has been read.
func (vr *VerifyingReader) Digest() DigestInfo {
	return DigestInfo{
		Algorithm: "sha256",
		Hex:       fmt.Sprintf("%x", vr.hash.Sum(nil)),
	}
}

// Size returns the number of bytes read.
func (vr *VerifyingReader) Size() int64 {
	return vr.size
}

// Verify checks that the computed digest matches the expected digest.
func (vr *VerifyingReader) Verify(expected DigestInfo) error {
	computed := vr.Digest()
	if computed.Algorithm != expected.Algorithm || computed.Hex != expected.Hex {
		return fmt.Errorf("%w: expected %s, got %s", ErrDigestMismatch, expected.String(), computed.String())
	}
	return nil
}
