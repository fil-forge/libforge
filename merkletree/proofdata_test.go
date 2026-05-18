package merkletree_test

import (
	"bytes"
	"testing"

	"github.com/fil-forge/libforge/merkletree"
	"github.com/stretchr/testify/require"
)

func sampleProof() merkletree.ProofData {
	pd := merkletree.ProofData{Index: 7}
	for i := 0; i < 4; i++ {
		var n merkletree.Node
		for j := range n {
			n[j] = byte(i*0x10 + j)
		}
		pd.Path = append(pd.Path, n)
	}
	return pd
}

func TestProofData_CBORRoundtrip(t *testing.T) {
	in := sampleProof()
	var buf bytes.Buffer
	require.NoError(t, in.MarshalCBOR(&buf))

	var out merkletree.ProofData
	require.NoError(t, out.UnmarshalCBOR(&buf))

	require.Equal(t, in, out)
}

func TestProofData_DagJSONRoundtrip(t *testing.T) {
	in := sampleProof()
	var buf bytes.Buffer
	require.NoError(t, in.MarshalDagJSON(&buf))

	var out merkletree.ProofData
	require.NoError(t, out.UnmarshalDagJSON(&buf))

	require.Equal(t, in, out)
}

func TestProofData_EmptyPathRoundtrip(t *testing.T) {
	in := merkletree.ProofData{Index: 0}

	var cb bytes.Buffer
	require.NoError(t, in.MarshalCBOR(&cb))
	var outCBOR merkletree.ProofData
	require.NoError(t, outCBOR.UnmarshalCBOR(&cb))
	require.Equal(t, in.Index, outCBOR.Index)
	require.Empty(t, outCBOR.Path)

	var jb bytes.Buffer
	require.NoError(t, in.MarshalDagJSON(&jb))
	var outJSON merkletree.ProofData
	require.NoError(t, outJSON.UnmarshalDagJSON(&jb))
	require.Equal(t, in.Index, outJSON.Index)
	require.Empty(t, outJSON.Path)
}
