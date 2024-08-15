package core

import (
	"blockchain/util"
	"bytes"
	"crypto/sha256"
	"fmt"
	"math"
	"math/big"
)

type ProofOfWork struct {
	block  *Block
	target *big.Int
}

// TargetBits is difficulty to mine a Block in POW
const TargetBits = 6
const MaxNonce = math.MaxInt

// NewProofOfWork builds a new ProofOfWork
func NewProofOfWork(b *Block) *ProofOfWork {
	target := big.NewInt(1)
	target.Lsh(target, uint(256-TargetBits))
	return &ProofOfWork{b, target}
}

// prepareData prepares Data to calculate Hash in order to get Nonce
func (pow *ProofOfWork) prepareData(nonce int) []byte {
	data := bytes.Join(
		[][]byte{
			[]byte(pow.block.PrevHash),
			pow.block.HashTransactions(),
			util.IntToHex(int64(pow.block.TimeStamp)),
			util.IntToHex(int64(nonce)),
		},
		[]byte{},
	)
	return data
}

// Run compares target and hashed data and mine block.
// If calculated hash is smaller than target, the process terminates
func (pow *ProofOfWork) Run() (int, []byte) {
	var hashInt big.Int
	var hash [32]byte
	nonce := 0
	for nonce < MaxNonce {
		data := pow.prepareData(nonce)
		hash = sha256.Sum256(data)
		fmt.Printf("\r%x", hash)

		hashInt.SetBytes(hash[:])
		if hashInt.Cmp(pow.target) == -1 {
			fmt.Println()
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

	hash := sha256.Sum256(
		pow.prepareData(pow.block.Nonce),
	)
	hashInt.SetBytes(hash[:])

	isValid := hashInt.Cmp(pow.target) == -1
	return isValid
}

// HashTransactions hashes transactions
func (b *Block) HashTransactions() []byte {
	var txHashes [][]byte
	var txHash [32]byte

	for _, tx := range b.Transactions {
		txHashes = append(txHashes, tx.GetHash())
	}
	txHash = sha256.Sum256(bytes.Join(txHashes, []byte{}))
	return txHash[:]
}
