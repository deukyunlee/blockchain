package core

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/go-playground/validator"
	"math/big"
	"sync"
	"time"
)

type block struct {
	number    big.Int   `validate:"required"`
	timeStamp time.Time `validate:"required"`
	hash      string    `validate:"required"`
	prevHash  string    `validate:"required"`
	data      string    `validate:"required"`
}

type Blockchain struct {
	blocks []*block
}

var bc *Blockchain
var once sync.Once
var errNotValid = errors.New("can't add this block")

func (bc *Blockchain) validateStructure(newBlock block) error {
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

func generateGenesisBlock() {
	once.Do(func() {
		bc = &Blockchain{}
		bc.AddBlock("Genesis Block")
	})
}

func (bc *Blockchain) AddBlock(data string) {
	prevHash := bc.getPrevHash()
	blockNo := bc.getBlockNumber()
	newBlock := newBlock(blockNo, data, prevHash)

	isValidated := bc.validateStructure(*newBlock)

	if isValidated != nil {
		fmt.Println("Block validation failed!")
	} else {
		bc.blocks = append(GetBlockchain().blocks, newBlock)
	}
}

func newBlock(blockNo big.Int, data string, prevHash string) *block {
	newBlock := &block{blockNo, time.Now(), "", prevHash, data}
	newBlock.calculateHash()
	return newBlock
}

func (b *block) calculateHash() {
	hash := sha256.Sum256([]byte(b.data + b.prevHash))
	b.hash = hex.EncodeToString(hash[:])
}

func GetBlockchain() *Blockchain {
	if bc == nil {
		generateGenesisBlock()
	}
	return bc
}

func (bc *Blockchain) getPrevHash() string {
	if len(GetBlockchain().blocks) > 0 {
		return GetBlockchain().blocks[len(GetBlockchain().blocks)-1].hash
	}
	return "First Block"
}

func (bc *Blockchain) getBlockNumber() big.Int {
	if len(GetBlockchain().blocks) > 0 {
		prevBlockNo := GetBlockchain().blocks[len(GetBlockchain().blocks)-1].number
		var nextBlockNo big.Int
		nextBlockNo.Add(&prevBlockNo, big.NewInt(1))
		return nextBlockNo
	}
	return *big.NewInt(1)
}

func (bc *Blockchain) ShowBlocks() {
	for _, block := range GetBlockchain().blocks {
		fmt.Printf("blockNo: %v\n", block.number.String())
		fmt.Printf("TimeStamp: %v\n", block.timeStamp)
		fmt.Printf("Data: %s\n", block.data)
		fmt.Printf("Hash: %s\n", block.hash)
		fmt.Printf("Prev Hash: %s\n", block.prevHash)
	}
}
