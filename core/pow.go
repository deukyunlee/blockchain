package core

import (
	"blockchain/util"
	"bytes"
	"crypto/sha256"
	"fmt"
	"math"
	"math/big"
)

// TargetBits is difficulty to mine a Block in POW
const TargetBits = 6
const MaxNonce = math.MaxUint64

// NewProofOfWork builds a new ProofOfWork
func NewProofOfWork(b *Block) *ProofOfWork {
	target := big.NewInt(1)
	target.Lsh(target, uint(256-TargetBits))
	return &ProofOfWork{b, target}
}

// prepareData prepares Data to calculate Hash in order to get Nonce
func (pow *ProofOfWork) prepareData(nonce uint64) []byte {
	return bytes.Join(
		[][]byte{
			[]byte(pow.Block.PrevHash),
			[]byte(pow.Block.Data),
			util.UintToHex(uint64(pow.Block.TimeStamp.UnixNano())),
			util.UintToHex(TargetBits),
			util.UintToHex(nonce),
		}, []byte{},
	)
}

// Run compares target and hashed data and mine block.
// If calculated hash is smaller than target, the process terminates
func (pow *ProofOfWork) Run() (uint64, []byte) {
	var hashInt big.Int
	var hash [32]byte

	nonce := uint64(0)

	for nonce < MaxNonce {
		data := pow.prepareData(nonce)
		hash = sha256.Sum256(data)

		fmt.Printf("\r%x", hash)

		hashInt.SetBytes(hash[:])
		if hashInt.Cmp(pow.Target) == -1 {
			break
		} else {
			nonce++
		}
	}

	return nonce, hash[:]
}

// Validate checks if certain block is mined through POW or not
func (pow *ProofOfWork) Validate() bool {
	var hashInt big.Int

	hash := sha256.Sum256(pow.prepareData(pow.Block.Nonce))
	hashInt.SetBytes(hash[:])

	isValid := hashInt.Cmp(pow.Target) == -1

	return isValid
}
