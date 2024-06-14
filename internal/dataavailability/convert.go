package dataavailability

import (
	"math/big"

	shareloader "github.com/calindra/nonodo/internal/dataavailability/contracts/ShareLoader.sol"

	"github.com/tendermint/tendermint/crypto/merkle"
	"github.com/tendermint/tendermint/libs/bytes"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
)

// Methods for converting for use with DAVerifier Library
// See https://docs.celestia.org/developers/blobstream-proof-queries#converting-the-proofs-to-be-usable-in-the-daverifier-library

func toNamespaceMerkleMultiProofs(proofs []*tmproto.NMTProof) []shareloader.NamespaceMerkleMultiproof {
	shareProofs := make([]shareloader.NamespaceMerkleMultiproof, len(proofs))
	for i, proof := range proofs {
		sideNodes := make([]shareloader.NamespaceNode, len(proof.Nodes))
		for j, node := range proof.Nodes {
			sideNodes[j] = *toNamespaceNode(node)
		}
		shareProofs[i] = shareloader.NamespaceMerkleMultiproof{
			BeginKey:  big.NewInt(int64(proof.Start)),
			EndKey:    big.NewInt(int64(proof.End)),
			SideNodes: sideNodes,
		}
	}
	return shareProofs
}

func minNamespace(innerNode []byte) *shareloader.Namespace {
	version := innerNode[0]
	var id [28]byte
	for i, b := range innerNode[1:29] {
		id[i] = b
	}
	return &shareloader.Namespace{
		Version: [1]byte{version},
		Id:      id,
	}
}

func maxNamespace(innerNode []byte) *shareloader.Namespace {
	version := innerNode[29]
	var id [28]byte
	for i, b := range innerNode[30:58] {
		id[i] = b
	}
	return &shareloader.Namespace{
		Version: [1]byte{version},
		Id:      id,
	}
}

func toNamespaceNode(node []byte) *shareloader.NamespaceNode {
	minNs := minNamespace(node)
	maxNs := maxNamespace(node)
	var digest [32]byte
	for i, b := range node[58:] {
		digest[i] = b
	}
	return &shareloader.NamespaceNode{
		Min:    *minNs,
		Max:    *maxNs,
		Digest: digest,
	}
}

func namespace(namespaceID []byte, version uint8) *shareloader.Namespace {
	var id [28]byte
	copy(id[:], namespaceID)
	return &shareloader.Namespace{
		Version: [1]byte{version},
		Id:      id,
	}
}

func toRowRoots(roots []bytes.HexBytes) []shareloader.NamespaceNode {
	rowRoots := make([]shareloader.NamespaceNode, len(roots))
	for i, root := range roots {
		rowRoots[i] = *toNamespaceNode(root.Bytes())
	}
	return rowRoots
}

func toRowProofs(proofs []*merkle.Proof) []shareloader.BinaryMerkleProof {
	rowProofs := make([]shareloader.BinaryMerkleProof, len(proofs))
	for i, proof := range proofs {
		sideNodes := make([][32]byte, len(proof.Aunts))
		for j, sideNode := range proof.Aunts {
			var bzSideNode [32]byte
			for k, b := range sideNode {
				bzSideNode[k] = b
			}
			sideNodes[j] = bzSideNode
		}
		rowProofs[i] = shareloader.BinaryMerkleProof{
			SideNodes: sideNodes,
			Key:       big.NewInt(proof.Index),
			NumLeaves: big.NewInt(proof.Total),
		}
	}
	return rowProofs
}

func toAttestationProof(
	nonce uint64,
	height uint64,
	blockDataRoot [32]byte,
	dataRootInclusionProof merkle.Proof,
) shareloader.AttestationProof {
	sideNodes := make([][32]byte, len(dataRootInclusionProof.Aunts))
	for i, sideNode := range dataRootInclusionProof.Aunts {
		var bzSideNode [32]byte
		for k, b := range sideNode {
			bzSideNode[k] = b
		}
		sideNodes[i] = bzSideNode
	}

	return shareloader.AttestationProof{
		TupleRootNonce: big.NewInt(int64(nonce)),
		Tuple: shareloader.DataRootTuple{
			Height:   big.NewInt(int64(height)),
			DataRoot: blockDataRoot,
		},
		Proof: shareloader.BinaryMerkleProof{
			SideNodes: sideNodes,
			Key:       big.NewInt(dataRootInclusionProof.Index),
			NumLeaves: big.NewInt(dataRootInclusionProof.Total),
		},
	}
}
