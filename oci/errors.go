package oci

import "errors"

var (
	// ErrBlobNotFound is returned when a requested blob does not exist.
	ErrBlobNotFound = errors.New("blob not found")

	// ErrManifestNotFound is returned when a requested manifest does not exist.
	ErrManifestNotFound = errors.New("manifest not found")

	// ErrUploadNotFound is returned when an upload session does not exist.
	ErrUploadNotFound = errors.New("upload not found")

	// ErrDigestMismatch is returned when computed digest doesn't match expected.
	ErrDigestMismatch = errors.New("digest mismatch")

	// ErrInvalidDigest is returned when a digest string is malformed.
	ErrInvalidDigest = errors.New("invalid digest")

	// ErrManifestTooLarge is returned when a manifest exceeds the max size.
	ErrManifestTooLarge = errors.New("manifest too large")
)
