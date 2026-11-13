// Package cpepack includes an opt-in research-only verifier for an independently
// expected selected subset. No signature, source trust root, semantic acceptance
// or external-byte approval is established by this function.
package cpepack

// BuildPinnedSubset verifies a research semantic source and an explicit
// representation selection against two separately admitted expected SHA-256
// hex digests. The expected values MUST come from an independent authorized
// release/consumer lock, never from the same untrusted candidate view.
// This function does not read or authorize bytes and does not promote concepts.
func BuildPinnedSubset(
	packages []Package,
	lock Lock,
	packageRoots []string,
	selector SubsetSelector,
	refs []RepresentationRef,
	public bool,
	expectedSourceLockSHA256 string,
	expectedSelectionSHA256 string,
) (SubsetView, error) {
	if !isHex(expectedSourceLockSHA256, 64) || !isHex(expectedSelectionSHA256, 64) {
		return SubsetView{}, fail("missing_or_invalid_independent_pin")
	}
	// Keep the existing research package content, schema, source and authority
	// verification as the sole entrance. Do not accept caller-supplied views.
	view, err := BuildSubset(packages, lock, packageRoots, selector, refs, public)
	if err != nil {
		return SubsetView{}, err
	}
	if view.SourceLockSHA256 != expectedSourceLockSHA256 {
		return SubsetView{}, fail("source_pin_mismatch")
	}
	if view.Digest != expectedSelectionSHA256 {
		return SubsetView{}, fail("selection_pin_mismatch")
	}
	return view, nil
}
