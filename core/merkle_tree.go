package core

import "crypto/sha256"

type MerkleTree struct {
	Root       *Node
	merkleRoot []byte
	Leafs      []*Node
}

type Node struct {
	Tree   *MerkleTree
	Parent *Node
	Left   *Node
	Right  *Node
	Hash   []byte
}

func NewMerkleTree(data [][]byte) *MerkleTree {
	var nodes []*Node

	// Make it even
	if len(data)%2 != 0 {
		data = append(data, data[len(data)-1])
	}

	// Hash transactions
	for _, tx := range data {
		node := NewMerkleNode(nil, nil, tx)
		nodes = append(nodes, node)
	}

	for i := 0; i < len(data)/2; i++ {
		var newLevel []*Node

		for j := 0; j < len(nodes); j += 2 {
			node := NewMerkleNode(nodes[j], nodes[j+1], nil)
			newLevel = append(newLevel, node)
		}

		nodes = newLevel
	}

	mTree := MerkleTree{nodes[0], nodes[0].Hash, nodes}
	return &mTree
}

func NewMerkleNode(left, right *Node, data []byte) *Node {
	mNode := Node{}

	if left == nil && right == nil {
		hash := sha256.Sum256(data)
		mNode.Hash = hash[:]
	} else {
		hashes := append(left.Hash, right.Hash...)
		hash := sha256.Sum256(hashes)
		mNode.Hash = hash[:]
		left.Parent = &mNode
		right.Parent = &mNode
	}

	mNode.Left = left
	mNode.Right = right

	return &mNode
}
