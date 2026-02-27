package oci

import "path"

// BlobDataPath returns the storage path for a blob's data file.
// Layout: v2/blobs/<algorithm>/<first-2-hex>/<full-hex>/data
func BlobDataPath(d DigestInfo) string {
	return path.Join("v2/blobs", d.Algorithm, d.ShortHex(), d.Hex, "data")
}

// ManifestRevisionLinkPath returns the storage path for a manifest revision link.
// Layout: v2/repositories/<name>/_manifests/revisions/<algorithm>/<hex>/link
func ManifestRevisionLinkPath(name string, d DigestInfo) string {
	return path.Join("v2/repositories", name, "_manifests/revisions", d.Algorithm, d.Hex, "link")
}

// ManifestTagCurrentLinkPath returns the storage path for a tag's current link.
// Layout: v2/repositories/<name>/_manifests/tags/<tag>/current/link
func ManifestTagCurrentLinkPath(name, tag string) string {
	return path.Join("v2/repositories", name, "_manifests/tags", tag, "current/link")
}

// ManifestTagsDir returns the storage path for the tags directory of a repository.
// Layout: v2/repositories/<name>/_manifests/tags
func ManifestTagsDir(name string) string {
	return path.Join("v2/repositories", name, "_manifests/tags")
}

// UploadDataPath returns the storage path for an in-progress upload's data.
// Layout: v2/uploads/<uuid>/data
func UploadDataPath(uuid string) string {
	return path.Join("v2/uploads", uuid, "data")
}
