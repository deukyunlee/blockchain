package core

import (
	"errors"
	"fmt"
	"github.com/go-playground/validator"
	"math/big"
	"strconv"
	"sync"
	"time"
)

type Block struct {
	Number    big.Int   `validate:"required"`
	TimeStamp time.Time `validate:"required"`
	Hash      []byte    `validate:"required"`
	PrevHash  []byte    `validate:"required"`
	Data      string    `validate:"required"`
	Nonce     uint64    `validate:"required"`
}

type Blockchain struct {
	Blocks []*Block
}

type ProofOfWork struct {
	Block  *Block
	Target *big.Int
}

var bc *Blockchain
var once sync.Once
var errNotValid = errors.New("can't add this Block")

const InitialNonce = uint64(0)

// validateStructure validates Block struct
func (bc *Blockchain) validateStructure(newBlock Block) error {
	fmt.Println(newBlock)
	validate := validator.New()

	err := validate.Struct(newBlock)
	if err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			fmt.Println(err)
		}
		return errNotValid
	}
	return nil
}

// generateGenesisBlock creates Genesis Block only once
func generateGenesisBlock() {
	once.Do(func() {
		bc = &Blockchain{}
		bc.AddBlock("Genesis Block")
		time.Sleep(1 * time.Second)
	})
}

// AddBlock appends new Block to Blocks only if struct is validated
func (bc *Blockchain) AddBlock(data string) {
	prevHash := bc.getPrevHash()
	blockNo := bc.getBlockNumber()
	newBlock := newBlock(blockNo, data, prevHash)

	isValidated := bc.validateStructure(*newBlock)

	if isValidated != nil {
		fmt.Println("Block validation failed!")
	} else {
		bc.Blocks = append(GetBlockchain().Blocks, newBlock)
	}
}

// newBlock calculates Hash and create new struct
func newBlock(blockNo big.Int, data string, prevHash []byte) *Block {
	block := &Block{blockNo, time.Now(), []byte{}, prevHash, data, InitialNonce}
	//newBlock.calculateHash()
	pow := NewProofOfWork(block)

	block.Nonce, block.Hash = pow.Run()
	return block
}

//// calculateHash calculates hash using sha256
//func (b *Block) calculateHash() {
//	hash := sha256.Sum256([]byte(b.Data + b.PrevHash))
//	b.Hash = hash[:]
//}

// GetBlockchain returns BlockChain struct which is list of Blocks, and generateGenesisBlock if there is no Data
func GetBlockchain() *Blockchain {
	if bc == nil {
		generateGenesisBlock()
	}
	return bc
}

// getBlockNumber returns previous blockHash
func (bc *Blockchain) getPrevHash() []byte {
	if len(GetBlockchain().Blocks) > 0 {
		return GetBlockchain().Blocks[len(GetBlockchain().Blocks)-1].Hash
	}
	return []byte("First Block")
}

// getBlockNumber returns current blockNo
func (bc *Blockchain) getBlockNumber() big.Int {
	if len(GetBlockchain().Blocks) > 0 {
		prevBlockNo := GetBlockchain().Blocks[len(GetBlockchain().Blocks)-1].Number
		var nextBlockNo big.Int
		nextBlockNo.Add(&prevBlockNo, big.NewInt(1))
		return nextBlockNo
	}
	return *big.NewInt(1)
}

// ShowBlocks shows blockData in Block
func (bc *Blockchain) ShowBlocks() {
	for _, block := range GetBlockchain().Blocks {
		pow := NewProofOfWork(block)
		fmt.Printf("blockNo: %v\n", block.Number.String())
		fmt.Printf("TimeStamp: %v\n", block.TimeStamp)
		fmt.Printf("Data: %s\n", block.Data)
		fmt.Printf("Hash: %x\n", block.Hash)
		fmt.Printf("Prev Hash: %x\n", block.PrevHash)
		fmt.Printf("Nonce: %d\n", block.Nonce)
		fmt.Printf("is Validated: %s\n", strconv.FormatBool(pow.Validate()))
	}
}
