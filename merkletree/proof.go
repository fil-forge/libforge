package merkletree

// ProofData is a Merkle inclusion proof: the sibling nodes along the path from
// a leaf (or subtree root) up to the tree root, plus the leaf's index within
// the leaf level. Each entry in Path MUST be NodeSize (32) bytes.
type ProofData struct {
	// Path is the sequence of sibling node digests from the leaf level up to
	// the root. len(Path) equals the depth of the tree.
	Path [][]byte `cborgen:"path" dagjsongen:"path"`
	// Index is the 0-based position of the leaf (or subtree) within its level.
	// The leftmost leaf has index 0.
	Index uint64 `cborgen:"index" dagjsongen:"index"`
}

// Depth returns the depth of the tree the proof validates against.
func (p ProofData) Depth() int { return len(p.Path) }
