// Package cpepack: research-only representation references and deterministic
// subsets. No artifact download, authority promotion or byte custody occurs here.
package cpepack

import (
	"encoding/json"
	"mime"
	"sort"
	"strings"
)

const (
	RepresentationVersion = "cpe-representation-ref-research-v0"
	SubsetVersion         = "cpe-subset-research-v0"
)

// ArtifactIdentity is a typed external identity, not a claim that bytes are
// available, verified, safe, licensed, or approved for a consumer.
// In particular, an OCI bundle digest is not an individual file SHA-256.
type ArtifactIdentity struct {
	Kind   string `json:"kind"` // oci, content-manifest, raw-sha256
	Digest string `json:"digest"`
}

// RepresentationRef associates an example, counterexample or parameterization
// with a concept without merging the artifact identity into semantic identity.
// No URL, credential, local path, personal identity or cache state is carried.
type RepresentationRef struct {
	SchemaVersion  string           `json:"schema_version"`
	ReferenceID    string           `json:"reference_id"`
	ConceptID      string           `json:"concept_id"`
	Role           string           `json:"role"`
	Modality       string           `json:"modality"`
	MediaType      string           `json:"media_type"`
	FormatVersion  string           `json:"format_version"`
	Artifact       ArtifactIdentity `json:"artifact"`
	Visibility     string           `json:"visibility"` // public or private
	Rights         string           `json:"rights"`     // redistributable, restricted, unknown
	SourceRevision string           `json:"source_revision"`
	EvidenceIDs    []string         `json:"evidence_ids,omitempty"`
	Dependencies   []string         `json:"dependencies,omitempty"` // other representation IDs
}

// RequiredEdge is a consumer selection requirement. It must correspond to a
// pre-existing typed semantic relation. It does not create a new CPE relation.
type RequiredEdge struct {
	FromID    string `json:"from_id"`
	Predicate string `json:"predicate"`
	TargetID  string `json:"target_id"`
}

type SubsetSelector struct {
	SchemaVersion  string         `json:"schema_version"`
	ProfileID      string         `json:"profile_id"`
	RootConceptIDs []string       `json:"root_concept_ids"`
	RequiredEdges  []RequiredEdge `json:"required_edges,omitempty"`
}

type OutOfSubsetEdge struct {
	FromID    string `json:"from_id"`
	Predicate string `json:"predicate"`
	TargetID  string `json:"target_id"`
}

type SelectedRepresentation struct {
	RepresentationRef
	DependencyOnly bool `json:"dependency_only"`
}

// SubsetView is an ephemeral research projection, NOT an accepted CPE release.
// The digest binds the input package lock and normalized selector, concepts,
// external relation references, and representation references.
type SubsetView struct {
	SchemaVersion    string                   `json:"schema_version"`
	Authority        string                   `json:"authority"`
	SourceLockSHA256 string                   `json:"source_lock_sha256"`
	SelectorSHA256   string                   `json:"selector_sha256"`
	Concepts         []Concept                `json:"concepts"`
	OutOfSubsetEdges []OutOfSubsetEdge        `json:"out_of_subset_edges"`
	Representations  []SelectedRepresentation `json:"representations"`
	Digest           string                   `json:"digest"`
}

func canonicalSelector(s SubsetSelector) (SubsetSelector, error) {
	if s.SchemaVersion != SubsetVersion || !validID(s.ProfileID) || len(s.RootConceptIDs) == 0 {
		return SubsetSelector{}, fail("invalid_subset_selector")
	}
	roots, err := sortedUnique(s.RootConceptIDs)
	if err != nil {
		return SubsetSelector{}, fail("invalid_subset_roots")
	}
	s.RootConceptIDs = roots
	edges := append([]RequiredEdge{}, s.RequiredEdges...)
	sort.Slice(edges, func(i, j int) bool {
		a, b := edges[i], edges[j]
		if a.FromID != b.FromID {
			return a.FromID < b.FromID
		}
		if a.Predicate != b.Predicate {
			return a.Predicate < b.Predicate
		}
		return a.TargetID < b.TargetID
	})
	for i, e := range edges {
		if !validID(e.FromID) || !validID(e.Predicate) || !validID(e.TargetID) {
			return SubsetSelector{}, fail("invalid_required_edge")
		}
		if i > 0 && edges[i-1] == e {
			return SubsetSelector{}, fail("duplicate_required_edge")
		}
	}
	s.RequiredEdges = edges
	return s, nil
}

func validRepresentation(r RepresentationRef) error {
	if r.SchemaVersion != RepresentationVersion || !validID(r.ReferenceID) || !validID(r.ConceptID) {
		return fail("invalid_representation")
	}
	switch r.Role {
	case "demonstrates", "counterexample", "evidences", "parameterizes", "recognition_fixture":
	default:
		return fail("invalid_representation_role")
	}
	// These are bounded identifiers, not free-form URLs, local paths, or secrets.
	// Keep provider locators/credentials out of public-safe reference metadata.
	if !validID(r.Modality) || !validID(r.FormatVersion) || !validID(r.SourceRevision) {
		return fail("incomplete_representation_metadata")
	}
	mediaType, params, err := mime.ParseMediaType(r.MediaType)
	if len(r.MediaType) > 192 || err != nil || !strings.Contains(mediaType, "/") || len(params) != 0 || mediaType != r.MediaType {
		return fail("incomplete_representation_metadata")
	}
	if r.Artifact.Kind != "oci" && r.Artifact.Kind != "content-manifest" && r.Artifact.Kind != "raw-sha256" {
		return fail("invalid_artifact_identity_kind")
	}
	if !strings.HasPrefix(r.Artifact.Digest, "sha256:") || !isHex(strings.TrimPrefix(r.Artifact.Digest, "sha256:"), 64) {
		return fail("invalid_artifact_digest")
	}
	if r.Visibility != "public" && r.Visibility != "private" {
		return fail("invalid_representation_visibility")
	}
	if r.Rights != "redistributable" && r.Rights != "restricted" && r.Rights != "unknown" {
		return fail("invalid_representation_rights")
	}
	return nil
}

// BuildSubset enforces package content-digest, schema/identity, source pin and
// public package-closure checks through the existing research-only Build gate
// before selecting concepts or external example references. Git origin itself
// is still only syntactically pinned (see cpepack R1 research limitations).
func BuildSubset(packages []Package, lock Lock, packageRoots []string, selector SubsetSelector, refs []RepresentationRef, public bool) (SubsetView, error) {
	source, err := Build(packages, lock, packageRoots, public)
	if err != nil {
		return SubsetView{}, err
	}
	return materializeSubset(source, selector, refs, public)
}

// materializeSubset is internal: callers cannot supply an unverified source
// projection and claim it has passed the package lock. It neither fetches nor
// verifies artifact bytes. Public output rejects private/restricted refs.
func materializeSubset(source Projection, selector SubsetSelector, refs []RepresentationRef, public bool) (SubsetView, error) {
	empty := SubsetView{}
	if source.SchemaVersion != SnapshotVersion || source.Authority != "research_only" || !isHex(source.LockSHA256, 64) {
		return empty, fail("unqualified_source_projection")
	}
	selection, err := canonicalSelector(selector)
	if err != nil {
		return empty, err
	}
	byConcept := make(map[string]Concept, len(source.Concepts))
	for _, c := range source.Concepts {
		if !validID(c.ID) {
			return empty, fail("invalid_source_concept")
		}
		if _, ok := byConcept[c.ID]; ok {
			return empty, fail("duplicate_source_concept")
		}
		byConcept[c.ID] = c
	}
	requirements := make(map[string][]string)
	for _, edge := range selection.RequiredEdges {
		c, ok := byConcept[edge.FromID]
		if !ok {
			return empty, fail("unknown_required_edge_source")
		}
		matched := false
		for _, rel := range c.Relations {
			if rel.Predicate == edge.Predicate && rel.TargetID == edge.TargetID {
				matched = true
				break
			}
		}
		if !matched {
			return empty, fail("unproven_required_edge")
		}
		if _, ok := byConcept[edge.TargetID]; !ok {
			return empty, fail("unknown_required_edge_target")
		}
		requirements[edge.FromID] = append(requirements[edge.FromID], edge.TargetID)
	}
	selected := make(map[string]bool)
	var includeConcept func(string) error
	includeConcept = func(id string) error {
		if selected[id] {
			return nil
		}
		if _, ok := byConcept[id]; !ok {
			return fail("unknown_subset_root")
		}
		selected[id] = true
		for _, dep := range requirements[id] {
			if err := includeConcept(dep); err != nil {
				return err
			}
		}
		return nil
	}
	for _, id := range selection.RootConceptIDs {
		if err := includeConcept(id); err != nil {
			return empty, err
		}
	}

	out := SubsetView{
		SchemaVersion:    SubsetVersion,
		Authority:        "research_only",
		SourceLockSHA256: source.LockSHA256,
		Concepts:         []Concept{},
		OutOfSubsetEdges: []OutOfSubsetEdge{},
		Representations:  []SelectedRepresentation{},
	}
	for id := range selected {
		out.Concepts = append(out.Concepts, byConcept[id])
	}
	sort.Slice(out.Concepts, func(i, j int) bool { return out.Concepts[i].ID < out.Concepts[j].ID })
	for _, c := range out.Concepts {
		for _, rel := range c.Relations {
			if !selected[rel.TargetID] {
				out.OutOfSubsetEdges = append(out.OutOfSubsetEdges, OutOfSubsetEdge{c.ID, rel.Predicate, rel.TargetID})
			}
		}
	}
	sort.Slice(out.OutOfSubsetEdges, func(i, j int) bool {
		a, b := out.OutOfSubsetEdges[i], out.OutOfSubsetEdges[j]
		if a.FromID != b.FromID {
			return a.FromID < b.FromID
		}
		if a.Predicate != b.Predicate {
			return a.Predicate < b.Predicate
		}
		return a.TargetID < b.TargetID
	})

	byRef := make(map[string]RepresentationRef, len(refs))
	for _, r := range refs {
		if err := validRepresentation(r); err != nil {
			return empty, err
		}
		if _, ok := byRef[r.ReferenceID]; ok {
			return empty, fail("duplicate_representation_id")
		}
		if _, ok := byConcept[r.ConceptID]; !ok {
			return empty, fail("representation_unknown_concept")
		}
		// Canonicalize set-like evidence and dependency identity separately.
		r.EvidenceIDs, err = sortedUnique(r.EvidenceIDs)
		if err != nil {
			return empty, fail("invalid_evidence_reference")
		}
		r.Dependencies, err = sortedUnique(r.Dependencies)
		if err != nil {
			return empty, fail("invalid_representation_dependency")
		}
		byRef[r.ReferenceID] = r
	}
	includedRefs := map[string]bool{}
	visitState := map[string]uint8{}
	var includeRef func(string) error
	includeRef = func(id string) error {
		if visitState[id] == 1 {
			return fail("representation_dependency_cycle")
		}
		if visitState[id] == 2 {
			return nil
		}
		ref, ok := byRef[id]
		if !ok {
			return fail("representation_dependency_missing")
		}
		if public && ref.Visibility != "public" {
			return fail("private_representation_in_public_closure")
		}
		if public && ref.Rights == "restricted" {
			return fail("restricted_representation_in_public_closure")
		}
		visitState[id] = 1
		for _, dep := range ref.Dependencies {
			if err := includeRef(dep); err != nil {
				return err
			}
		}
		visitState[id] = 2
		includedRefs[id] = true
		return nil
	}
	for _, r := range refs {
		if selected[r.ConceptID] {
			if err := includeRef(r.ReferenceID); err != nil {
				return empty, err
			}
		}
	}
	for id := range includedRefs {
		r := byRef[id]
		out.Representations = append(out.Representations, SelectedRepresentation{RepresentationRef: r, DependencyOnly: !selected[r.ConceptID]})
	}
	sort.Slice(out.Representations, func(i, j int) bool { return out.Representations[i].ReferenceID < out.Representations[j].ReferenceID })
	selJSON, err := json.Marshal(selection)
	if err != nil {
		return empty, err
	}
	out.SelectorSHA256 = sha(selJSON)
	material, err := json.Marshal(out) // Digest is still empty, fixed struct order.
	if err != nil {
		return empty, err
	}
	out.Digest = sha(material)
	return out, nil
}
