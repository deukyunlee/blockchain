package core

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/go-playground/validator"
	"log"
	"strconv"
	"sync"
	"time"
)

type Block struct {
	TimeStamp int32  `validate:"required"`
	Hash      []byte `validate:"required"`
	PrevHash  []byte `validate:"required"`
	Data      []byte `validate:"required"`
	Nonce     int    `validate:"min=0"`
}

type Blockchain struct {
	Db   *bolt.DB
	last []byte
}

type BlockchainTmp struct {
	db          *bolt.DB
	currentHash []byte
}

type Transaction struct {
	ID    []byte
	Txin  []TXInput
	Txout []TXOutput
}

// TXInput is about Transaction Input
type TXInput struct {
	Txid      []byte // transaction id
	TxoutIdx  int    // referenced output index number
	ScriptSig string // Unlock script
}

// TXOutput is about Transaction Output
type TXOutput struct {
	Value        int    // how much
	ScriptPubKey string // To public key - Lock script
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
		log.Panic(err)
	}

	newBlock := NewBlock(data, lastHash)

	err = bc.Db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("blocks"))
		err := b.Put(newBlock.Hash, newBlock.Serialize())
		if err != nil {
			log.Panic(err)
		}

		err = b.Put([]byte("last"), newBlock.Hash)
		if err != nil {
			log.Panic(err)
		}

		bc.last = newBlock.Hash

		fmt.Println("Successfully Added")

		return nil
	})
	if err != nil {
		log.Panic(err)
	}
}

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
	// defer db.Close()
	err = db.Update(func(tx *bolt.Tx) error {
		bc := tx.Bucket([]byte("blocks"))
		if bc == nil {
			genesis := generateGenesis()
			fmt.Println("Generate Genesis block")
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

// ShowBlocks shows blockData in Block
func (bc Blockchain) ShowBlocks() {
	bcT := bc.Iterator()

	for {
		block := bcT.getNextBlock()
		pow := NewProofOfWork(block)

		fmt.Printf("TimeStamp: %d\n", block.TimeStamp)
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
	newblock := &Block{int32(time.Now().Unix()), nil, prevHash, []byte(data), 0}
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

// GetHash hashes Transaction and returns the hash
func (tx *Transaction) GetHash() []byte {
	var writer bytes.Buffer
	var hash [32]byte

	enc := gob.NewEncoder(&writer)

	err := enc.Encode(tx)
	if err != nil {
		log.Fatal(err)
	}

	hash = sha256.Sum256(writer.Bytes())

	return hash[:]
}
