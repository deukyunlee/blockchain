package core

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/btcsuite/btcutil/base58"
	"github.com/go-playground/validator"
	"log"
	"os"
	"strconv"
	"sync"
	"time"
)

type Block struct {
	TimeStamp    int32          `validate:"required"`
	Hash         []byte         `validate:"required"`
	PrevHash     []byte         `validate:"required"`
	Transactions []*Transaction `validate:"required"`
	Nonce        int            `validate:"min=0"`
}

type Blockchain struct {
	Db   *bolt.DB
	last []byte
}

type BlockchainIterator struct {
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

// AddBlock gets last block using view function, adds to blocks bucket
// and updates last bucket
func (bc *Blockchain) AddBlock(transactions []*Transaction) {
	var lastHash []byte

	err := bc.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("blocks"))
		lastHash = b.Get([]byte("last"))

		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	newBlock := NewBlock(transactions, lastHash)

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

func dbExists() bool {
	dbFile := fmt.Sprintf(dbFile, "0600")
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}

	return true
}

// GetBlockchain opens BoltDB which is written in file, with mode 0600
// in order to start a read-write transaction, use DB.Update()
// to start read-only transaction, you can use DB.View()
// Bucket is key/value collection in BoltDB
// every key needs to be unique
func GetBlockchain() *Blockchain {
	if !dbExists() {
		fmt.Println("There's no blockchain yet. Create one first.")
		os.Exit(1)
	}
	var last []byte

	dbFile := fmt.Sprintf(dbFile, "0600")
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		bc := tx.Bucket([]byte("blocks"))
		last = bc.Get([]byte("last"))

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
		block := bcT.GetNextBlock()
		pow := NewProofOfWork(block)

		fmt.Printf("TimeStamp: %d\n", block.TimeStamp)
		fmt.Printf("Transaction: %s\n", block.Transactions)
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

func (bc *Blockchain) Iterator() *BlockchainIterator {
	bcT := &BlockchainIterator{bc.Db, bc.last}

	return bcT
}

// NewBlock prepares new block
func NewBlock(transactions []*Transaction, prevHash []byte) *Block {
	newblock := &Block{int32(time.Now().Unix()), nil, prevHash, transactions, 0}
	pow := NewProofOfWork(newblock)
	nonce, hash := pow.Run()

	newblock.Hash = hash[:]
	newblock.Nonce = nonce
	return newblock
}

func (bct *BlockchainIterator) GetNextBlock() *Block {
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

func generateGenesis(tx *Transaction) *Block {
	return NewBlock([]*Transaction{tx}, []byte{})
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

// FindUnspentTxs returns a list of transactions containing unspent outputs
func (bc *Blockchain) FindUnspentTxs(publicKeyHash []byte) []*Transaction {
	var unspentTXs []*Transaction
	spentTXOs := make(map[string][]int)
	bcI := bc.Iterator()

	for {
		block := bcI.getNextBlock()

		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)

		Outputs:
			for outIndex, out := range tx.Vout {
				if spentTXOs[txID] != nil {
					for _, spentOut := range spentTXOs[txID] {
						if spentOut == outIndex {
							continue Outputs
						}
					}
				}

				if out.IsLockedWithKey(publicKeyHash) {
					unspentTXs = append(unspentTXs, tx)
					continue Outputs
				}
			}

			if !tx.IsCoinbase() {
				for _, in := range tx.Vin {
					if in.Unlock(publicKeyHash) {
						inTxID := hex.EncodeToString(in.Txid)
						spentTXOs[inTxID] = append(spentTXOs[inTxID], in.TxoutIdx)
					}
				}
			}
		}

		if len(block.PrevHash) == 0 {
			break
		}
	}
	return unspentTXs
}

// FindUTXOs finds and returns unspent transaction outputs for the address
func (bc *Blockchain) FindUTXOs(publicKeyHash []byte, amount int) (int, map[string][]int) {
	unspentOutputs := make(map[string][]int)
	unspentTXs := bc.FindUnspentTxs(publicKeyHash)
	accumulated := 0

Work:
	for _, tx := range unspentTXs {
		txID := hex.EncodeToString(tx.ID)

		for index, txout := range tx.Vout {
			if txout.IsLockedWithKey(publicKeyHash) && accumulated < amount {
				accumulated += txout.Value
				unspentOutputs[txID] = append(unspentOutputs[txID], index)
				if accumulated >= amount {
					break Work
				}
			}
		}
	}

	return accumulated, unspentOutputs
}

func (bcI *BlockchainIterator) getNextBlock() *Block {
	var block *Block

	err := bcI.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("blocks"))
		encodedBlock := b.Get(bcI.currentHash)
		block = DeserializeBlock(encodedBlock)

		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	bcI.currentHash = block.PrevHash
	return block
}

// GetTransaction gets transaction
func (bc *Blockchain) GetTransaction(id []byte) (Transaction, error) {
	bcI := bc.Iterator()
	for {
		block := bcI.getNextBlock()
		for _, tx := range block.Transactions {
			if bytes.Equal(tx.ID, id) {
				return *tx, nil
			}
		}
		if len(block.PrevHash) == 0 {
			break
		}
	}
	return Transaction{}, errors.New("Transaction not found")
}

// SignTransaction signs inputs of a Transaction
func (bc *Blockchain) SignTransaction(tx *Transaction, privKey ecdsa.PrivateKey) {
	prevTXs := make(map[string]Transaction)

	for _, vin := range tx.Vin {
		prevTX, err := bc.GetTransaction(vin.Txid)
		if err != nil {
			log.Panic(err)
		}
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}

	tx.Sign(privKey, prevTXs)
}

func isValidWallet(address string) bool {
	_, _, err := base58.CheckDecode(address)

	return err == nil
}

func CreateBlockchain(address string) *Blockchain {
	if dbExists() {
		fmt.Println("Blockchain already exists.")
		os.Exit(1)
	}
	if !isValidWallet(address) {
		fmt.Println("Use correct wallet")
		os.Exit(1)
	}
	var last []byte
	dbFile := fmt.Sprintf(dbFile, "0600")
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Panic(err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		cb := NewCoinbaseTX(address, "init base")
		genesis := generateGenesis(cb)
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
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	bc := Blockchain{db, last}
	return &bc
}
