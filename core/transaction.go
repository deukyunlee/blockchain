package core

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"github.com/btcsuite/btcutil/base58"
	"log"
	"math/big"
)

type Transaction struct {
	ID   []byte
	Vin  []TXInput
	Vout []TXOutput
}

type ScriptSig struct {
	Signature []byte
	PublicKey []byte
}

type TXInput struct {
	Txid      []byte
	TxoutIdx  int
	ScriptSig *ScriptSig
}

type TXOutput struct {
	Value        int
	ScriptPubKey []byte
}

// NewUTXOTransaction creates a new transaction
func NewUTXOTransaction(from, to string, amount int, bc *Blockchain) *Transaction {
	var inputs []TXInput
	var outputs []TXOutput

	wallets, err := NewWallets()
	if err != nil {
		log.Panic(err)
	}
	wallet := wallets.GetWallet(from)
	publicKeyHash := HashPublicKey(wallet.PublicKey)
	balance, validOutputs := bc.FindUTXOs(publicKeyHash, amount)

	if balance < amount {
		log.Panic("ERROR: Not enough funds")
	}

	// Build a list of inputs
	for txid, outs := range validOutputs {
		for _, out := range outs {
			txID, err := hex.DecodeString(txid)
			if err != nil {
				log.Panic(err)
			}
			input := TXInput{txID, out, &ScriptSig{nil, wallet.PublicKey}}
			inputs = append(inputs, input)
		}
	}

	// Build a list of outputs
	outputs = append(outputs, *NewTXOutput(amount, to))
	if balance > amount {
		outputs = append(outputs, *NewTXOutput(balance-amount, from))
	}

	tx := Transaction{nil, inputs, outputs}
	tx.SetID()
	bc.SignTransaction(&tx, wallet.PrivateKey)

	return &tx
}

// IsCoinbase checks whether the transaction is coinbase
func (tx Transaction) IsCoinbase() bool {
	return len(tx.Vin) == 1 && len(tx.Vin[0].Txid) == 0 && tx.Vin[0].TxoutIdx == -1
}

// SetID sets ID of a transaction
func (tx *Transaction) SetID() {
	var encoded bytes.Buffer
	var hash [32]byte

	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(tx)
	if err != nil {
		log.Panic(err)
	}
	hash = sha256.Sum256(encoded.Bytes())

	tx.ID = hash[:]
}

// Creates a abbreviated copy of Transaction to use in sign
func (tx *Transaction) AbbreviatedCopy() Transaction {
	var inputs []TXInput

	// The public key stored in the input doesn't need to be signed.
	for _, vin := range tx.Vin {
		inputs = append(inputs, TXInput{vin.Txid, vin.TxoutIdx, nil})
	}

	abbreviatedTx := Transaction{tx.ID, inputs, tx.Vout}

	return abbreviatedTx
}

// Signs each input of a Transaction
func (tx *Transaction) Sign(privKey ecdsa.PrivateKey, prevTXs map[string]Transaction) {
	if tx.IsCoinbase() {
		return
	}

	for _, vin := range tx.Vin {
		if prevTXs[hex.EncodeToString(vin.Txid)].ID == nil {
			log.Panic("Error with Previous transaction")
		}
	}

	abbreviatedTx := tx.AbbreviatedCopy()

	for inId, vin := range abbreviatedTx.Vin {
		prevTx := prevTXs[hex.EncodeToString(vin.Txid)]

		abbreviatedTx.Vin[inId].ScriptSig = &ScriptSig{}
		abbreviatedTx.Vin[inId].ScriptSig.PublicKey = prevTx.Vout[vin.TxoutIdx].ScriptPubKey

		// Use ECDSA(not RSA)
		r, s, err := ecdsa.Sign(rand.Reader, &privKey, abbreviatedTx.ID)
		if err != nil {
			log.Panic(err)
		}
		signature := append(r.Bytes(), s.Bytes()...)

		tx.Vin[inId].ScriptSig.Signature = signature
		abbreviatedTx.Vin[inId].ScriptSig.PublicKey = nil
	}
}

// Verifies signatures of Transaction inputs
func (tx *Transaction) Verify(prevTXs map[string]Transaction) bool {
	if tx.IsCoinbase() {
		return true
	}

	for _, vin := range tx.Vin {
		if prevTXs[hex.EncodeToString(vin.Txid)].ID == nil {
			log.Panic("Error with Previous transaction")
		}
	}

	abbreviatedTx := tx.AbbreviatedCopy()
	curve := elliptic.P256() // The same curve used to generate key pairs.

	for inId, vin := range tx.Vin {
		prevTx := prevTXs[hex.EncodeToString(vin.Txid)]

		abbreviatedTx.Vin[inId].ScriptSig = &ScriptSig{}
		abbreviatedTx.Vin[inId].ScriptSig.PublicKey = prevTx.Vout[vin.TxoutIdx].ScriptPubKey

		sigLen := len(vin.ScriptSig.Signature)
		keyLen := len(vin.ScriptSig.PublicKey)

		var r, s big.Int
		var x, y big.Int

		// Signature is a pair of numbers.
		r.SetBytes(vin.ScriptSig.Signature[:(sigLen / 2)])
		s.SetBytes(vin.ScriptSig.Signature[(sigLen / 2):])

		// PublicKey is a pair of coordinates.
		x.SetBytes(vin.ScriptSig.PublicKey[:(keyLen / 2)])
		y.SetBytes(vin.ScriptSig.PublicKey[(keyLen / 2):])

		rawPublicKey := &ecdsa.PublicKey{Curve: curve, X: &x, Y: &y}
		if !ecdsa.Verify(rawPublicKey, abbreviatedTx.ID, &r, &s) {
			return false
		}
		abbreviatedTx.Vin[inId].ScriptSig.PublicKey = nil
	}
	return true
}

// Unlock Tx
func (tI TXInput) Unlock(publicKeyHash []byte) bool {
	lockingHash := HashPublicKey(tI.ScriptSig.PublicKey)

	return bytes.Equal(lockingHash, publicKeyHash)
}

// Check key
func (tO TXOutput) IsLockedWithKey(publicKeyHash []byte) bool {
	return bytes.Equal(tO.ScriptPubKey, publicKeyHash)
}

// Lock with publicKey
func (tO *TXOutput) Lock(address string) {
	publicKeyHash, _, err := base58.CheckDecode(address)
	if err != nil {
		log.Panic(err)
	}
	tO.ScriptPubKey = publicKeyHash
}

// NewTXOutput creates a new TXOutput
func NewTXOutput(value int, address string) *TXOutput {
	txo := &TXOutput{value, nil}
	txo.Lock(address)

	return txo
}

// NewCoinbaseTX creates a new coinbase transaction
func NewCoinbaseTX(to, data string) *Transaction {
	if data == "Mining reward" {
		b := make([]byte, 10)
		_, err := rand.Read(b)
		if err != nil {
			log.Panic(err)
		}

		data = fmt.Sprintf("%x", b)
	}

	txin := TXInput{[]byte{}, -1, &ScriptSig{nil, []byte(data)}}
	txout := *NewTXOutput(subsidy, to)
	tx := Transaction{nil, []TXInput{txin}, []TXOutput{txout}}
	tx.SetID()

	return &tx
}

// SerializeTxs serializes TXOutputs
func SerializeTxs(outs []TXOutput) []byte {
	var writer bytes.Buffer

	enc := gob.NewEncoder(&writer)
	err := enc.Encode(outs)
	if err != nil {
		log.Panic(err)
	}

	return writer.Bytes()
}

// DeserializeTxs deserializes TXOutputs
func DeserializeTxs(data []byte) []TXOutput {
	var writer []TXOutput

	dec := gob.NewDecoder(bytes.NewReader(data))
	err := dec.Decode(&writer)
	if err != nil {
		log.Panic(err)
	}

	return writer
}
