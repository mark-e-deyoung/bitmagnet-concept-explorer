package cpepack

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	ociIndexType    = "application/vnd.oci.image.index.v1+json"
	ociManifestType = "application/vnd.oci.image.manifest.v1+json"
	ociEmptyType    = "application/vnd.oci.empty.v1+json"
	ociCPEType      = "application/vnd.semper.cpe.readmodel.v0+json"
)

type ociDescriptor struct {
	MediaType    string            `json:"mediaType"`
	Digest       string            `json:"digest"`
	Size         int               `json:"size"`
	ArtifactType string            `json:"artifactType,omitempty"`
	Annotations  map[string]string `json:"annotations,omitempty"`
}
type ociManifest struct {
	SchemaVersion int             `json:"schemaVersion"`
	MediaType     string          `json:"mediaType"`
	ArtifactType  string          `json:"artifactType"`
	Config        ociDescriptor   `json:"config"`
	Layers        []ociDescriptor `json:"layers"`
}
type ociIndex struct {
	SchemaVersion int             `json:"schemaVersion"`
	MediaType     string          `json:"mediaType"`
	Manifests     []ociDescriptor `json:"manifests"`
}

// WriteOCIArtifact materializes a non-executable, local OCI-layout directory.
// No registry is contacted; it refuses private transitive dependencies and an
// existing output path. The returned digest identifies the immutable manifest.
func WriteOCIArtifact(out string, packages []Package, lock Lock, roots []string) (string, error) {
	projection, err := Build(packages, lock, roots, true)
	if err != nil {
		return "", err
	}
	if projection.Authority != "research_only" || projection.SchemaVersion != SnapshotVersion {
		return "", errors.New("unqualified_oci_projection")
	}
	payload, err := json.Marshal(projection)
	if err != nil {
		return "", err
	}
	if len(payload) == 0 || len(payload) > 2<<20 {
		return "", errors.New("oci_projection_over_budget")
	}
	empty := []byte("{}")
	manifest := ociManifest{SchemaVersion: 2, MediaType: ociManifestType, ArtifactType: ociCPEType,
		Config: ociDescriptor{MediaType: ociEmptyType, Digest: "sha256:" + sha(empty), Size: len(empty)},
		Layers: []ociDescriptor{{MediaType: ociCPEType, Digest: "sha256:" + sha(payload), Size: len(payload)}},
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return "", err
	}
	manifestHash := sha(manifestBytes)
	indexBytes, err := json.Marshal(ociIndex{SchemaVersion: 2, MediaType: ociIndexType, Manifests: []ociDescriptor{{MediaType: ociManifestType, Digest: "sha256:" + manifestHash, Size: len(manifestBytes), ArtifactType: ociCPEType, Annotations: map[string]string{"org.opencontainers.image.ref.name": "research-only"}}}})
	if err != nil {
		return "", err
	}
	if out == "" || out == "." || out == "/" {
		return "", errors.New("unsafe_oci_output")
	}
	if err := os.Mkdir(out, 0700); err != nil {
		return "", fmt.Errorf("oci output must not exist: %w", err)
	}
	blobDir := filepath.Join(out, "blobs", "sha256")
	if err := os.MkdirAll(blobDir, 0700); err != nil {
		return "", err
	}
	blobs := [][]byte{empty, payload, manifestBytes}
	for _, data := range blobs {
		if err := os.WriteFile(filepath.Join(blobDir, sha(data)), data, 0600); err != nil {
			return "", err
		}
	}
	if err := os.WriteFile(filepath.Join(out, "oci-layout"), []byte(`{"imageLayoutVersion":"1.0.0"}`), 0600); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(out, "index.json"), indexBytes, 0600); err != nil {
		return "", err
	}
	return "sha256:" + manifestHash, nil
}
