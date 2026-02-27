package oci

import "testing"

func TestBlobDataPath(t *testing.T) {
	d := DigestInfo{Algorithm: "sha256", Hex: "abcdef1234567890"}
	got := BlobDataPath(d)
	want := "v2/blobs/sha256/ab/abcdef1234567890/data"
	if got != want {
		t.Errorf("BlobDataPath() = %q, want %q", got, want)
	}
}

func TestManifestRevisionLinkPath(t *testing.T) {
	d := DigestInfo{Algorithm: "sha256", Hex: "abcdef1234567890"}
	got := ManifestRevisionLinkPath("myrepo/myimage", d)
	want := "v2/repositories/myrepo/myimage/_manifests/revisions/sha256/abcdef1234567890/link"
	if got != want {
		t.Errorf("ManifestRevisionLinkPath() = %q, want %q", got, want)
	}
}

func TestManifestTagCurrentLinkPath(t *testing.T) {
	got := ManifestTagCurrentLinkPath("myrepo/myimage", "latest")
	want := "v2/repositories/myrepo/myimage/_manifests/tags/latest/current/link"
	if got != want {
		t.Errorf("ManifestTagCurrentLinkPath() = %q, want %q", got, want)
	}
}

func TestManifestTagsDir(t *testing.T) {
	got := ManifestTagsDir("myrepo/myimage")
	want := "v2/repositories/myrepo/myimage/_manifests/tags"
	if got != want {
		t.Errorf("ManifestTagsDir() = %q, want %q", got, want)
	}
}

func TestUploadDataPath(t *testing.T) {
	got := UploadDataPath("550e8400-e29b-41d4-a716-446655440000")
	want := "v2/uploads/550e8400-e29b-41d4-a716-446655440000/data"
	if got != want {
		t.Errorf("UploadDataPath() = %q, want %q", got, want)
	}
}
