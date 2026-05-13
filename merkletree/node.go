package merkletree

// NodeSize is the size in bytes of a single Merkle tree node digest.
const NodeSize = 32

// Node is a fixed-size 32-byte Merkle tree digest. Wire types use [][]byte for
// proof paths (see ProofData) because cborgen does not support slices of
// fixed-size arrays; callers needing the typed form can convert via
// NodesFromBytes / NodesToBytes.
type Node [NodeSize]byte

// NodesFromBytes converts a `[][]byte` (e.g. ProofData.Path) to []Node,
// returning an error if any element is not NodeSize bytes.
func NodesFromBytes(src [][]byte) ([]Node, error) {
	out := make([]Node, len(src))
	for i, b := range src {
		if len(b) != NodeSize {
			return nil, &SizeError{Index: i, Got: len(b)}
		}
		copy(out[i][:], b)
	}
	return out, nil
}

// NodesToBytes converts a []Node to a `[][]byte` suitable for ProofData.Path.
func NodesToBytes(src []Node) [][]byte {
	out := make([][]byte, len(src))
	for i, n := range src {
		b := make([]byte, NodeSize)
		copy(b, n[:])
		out[i] = b
	}
	return out
}

// SizeError reports a path entry whose byte length is not NodeSize.
type SizeError struct {
	Index int
	Got   int
}

func (e *SizeError) Error() string {
	return "merkletree: path entry has wrong size"
}
