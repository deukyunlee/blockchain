package core

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/go-playground/validator"
	"log"
	"math/big"
	"strconv"
	"sync"
	"time"
)

type Block struct {
	TimeStamp time.Time `validate:"required"`
	Hash      []byte    `validate:"required"`
	PrevHash  []byte    `validate:"required"`
	Data      []byte    `validate:"required"`
	Nonce     uint64    `validate:"required"`
}

type Blockchain struct {
	Db   *bolt.DB
	last []byte
}

type ProofOfWork struct {
	Block  *Block
	Target *big.Int
}

type BlockchainTmp struct {
	db          *bolt.DB
	currentHash []byte
}

var bc *Blockchain
var once sync.Once
var errNotValid = errors.New("can't add this Block")

const InitialNonce = uint64(0)

// subsidy is inflation of new coin
const subsidy = 10

const dbFile = "dukechain_%s.db"

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

// AddBlock gets last block using view function, adds to blocks bucket
// and updates last bucket
func (bc *Blockchain) AddBlock(data string) {

	var lastHash []byte
	err := bc.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("blocks"))
		lastHash = b.Get([]byte("last"))

		return nil
	})

	if err != nil {
		log.Fatal(err)
	}

	newBlock := NewBlock(data, lastHash)

	err = bc.Db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("blocks"))
		err := b.Put(newBlock.Hash, newBlock.Serialize())

		if err != nil {
			log.Fatal(err)
		}

		err = b.Put([]byte("last"), newBlock.Hash)
		if err != nil {
			log.Fatal(err)
		}

		bc.last = newBlock.Hash

		log.Println("Sucessfully Added")

		return nil
	})

	if err != nil {
		log.Fatal(err)
	}
}

//// newBlock calculates Hash and create new struct
//func newBlock(data string, prevHash []byte) *Block {
//	block := &Block{time.Now(), []byte{}, prevHash, data, InitialNonce}
//	//newBlock.calculateHash()
//	pow := NewProofOfWork(block)
//
//	block.Nonce, block.Hash = pow.Run()
//	return block
//}

//// calculateHash calculates hash using sha256
//func (b *Block) calculateHash() {
//	hash := sha256.Sum256([]byte(b.Data + b.PrevHash))
//	b.Hash = hash[:]
//}

// GetBlockchain opens BoltDB which is written in file, with mode 0600
// in order to start a read-write transaction, use DB.Update()
// to start read-only transaction, you can use DB.View()
// Bucket is key/value collection in BoltDB
// every key needs to be unique
func GetBlockchain() *Blockchain {
	var last []byte

	dbFile := fmt.Sprintf(dbFile, "0600")
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		bc := tx.Bucket([]byte("blocks"))
		if bc == nil {
			genesis := generateGenesis()
			log.Println("Generate Genesis block")

			b, err := tx.CreateBucket([]byte("blocks"))
			if err != nil {
				return err
			}
			err = b.Put(genesis.Hash, genesis.Serialize())
			if err != nil {
				return err
			}

			err = b.Put([]byte("last"), genesis.Hash)
			if err != nil {
				return err
			}

			last = genesis.Hash
		} else {
			last = bc.Get([]byte("last"))
		}
		return nil
	})

	if err != nil {
		log.Fatal(err)
	}

	bc := Blockchain{db, last}

	return &bc
}

//// getPrevHash returns previous blockHash
//func (bc *Blockchain) getPrevHash() []byte {
//	if len(GetBlockchain().Blocks) > 0 {
//		return GetBlockchain().Blocks[len(GetBlockchain().Blocks)-1].Hash
//	}
//	return []byte("First Block")
//}

//// getBlockNumber returns current blockNo
//func (bc *Blockchain) getBlockNumber() big.Int {
//	if len(GetBlockchain().Blocks) > 0 {
//		prevBlockNo := GetBlockchain().Blocks[len(GetBlockchain().Blocks)-1].Number
//		var nextBlockNo big.Int
//		nextBlockNo.Add(&prevBlockNo, big.NewInt(1))
//		return nextBlockNo
//	}
//	return *big.NewInt(1)
//}

// ShowBlocks shows blockData in Block
func (bc *Blockchain) ShowBlocks() {

	bcT := bc.Iterator()

	for {
		block := bcT.getNextBlock()
		pow := NewProofOfWork(block)

		fmt.Printf("TimeStamp: %v\n", block.TimeStamp)
		fmt.Printf("Data: %s\n", block.Data)
		fmt.Printf("Hash: %x\n", block.Hash)
		fmt.Printf("Prev Hash: %x\n", block.PrevHash)
		fmt.Printf("Nonce: %d\n", block.Nonce)
		fmt.Printf("is Validated: %s\n", strconv.FormatBool(pow.Validate()))

		if len(block.PrevHash) == 0 {
			break
		}
	}
}

// Serialize serializes blockData before sending
func (b *Block) Serialize() []byte {
	var value bytes.Buffer

	encoder := gob.NewEncoder(&value)
	err := encoder.Encode(b)
	if err != nil {
		log.Fatal("Encode Error: ", err)
	}

	return value.Bytes()
}

// DeserializeBlock deserializes bytes data into block
func DeserializeBlock(d []byte) *Block {
	var block Block

	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&block)
	if err != nil {
		log.Fatal("Decode Error: ", err)
	}

	return &block
}

func (bc *Blockchain) Iterator() *BlockchainTmp {
	bcT := &BlockchainTmp{bc.Db, bc.last}

	return bcT
}

// NewBlock prepares new block
func NewBlock(data string, prevHash []byte) *Block {
	newblock := &Block{time.Now(), nil, prevHash, []byte(data), 0}
	pow := NewProofOfWork(newblock)
	nonce, hash := pow.Run()

	newblock.Hash = hash[:]
	newblock.Nonce = nonce
	return newblock
}

func (bct *BlockchainTmp) getNextBlock() *Block {
	var block *Block

	err := bct.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("blocks"))
		encodedBlock := b.Get(bct.currentHash)
		block = DeserializeBlock(encodedBlock)

		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	bct.currentHash = block.PrevHash
	return block
}

func generateGenesis() *Block {
	return NewBlock("Genesis Block", []byte{})
}
