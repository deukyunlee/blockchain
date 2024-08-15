package core

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"github.com/btcsuite/btcutil/base58"
	"golang.org/x/crypto/ripemd160"
	"log"
	"os"
)

const version = byte(0x00)
const walletFile = "gowallet.dat"

type Wallet struct {
	PrivateKey ecdsa.PrivateKey
	PublicKey  []byte
}

type Wallets struct {
	Wallets map[string]*Wallet
}

// NewWallet generate New Wallet
func NewWallet() *Wallet {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Panic(err)
	}

	publicKey := append(privateKey.PublicKey.X.Bytes(), privateKey.PublicKey.Y.Bytes()...)

	return &Wallet{*privateKey, publicKey}
}

// HashPublicKey hashes public key
func HashPublicKey(pubKey []byte) []byte {
	publicSHA256 := sha256.Sum256(pubKey) // Public key를 SHA-256으로 해싱

	// RIPEMD-160으로 다시 해싱
	RIPEMD160Hasher := ripemd160.New()
	_, err := RIPEMD160Hasher.Write(publicSHA256[:])
	if err != nil {
		log.Panic(err)
	}

	publicRIPEMD160 := RIPEMD160Hasher.Sum(nil)
	return publicRIPEMD160
}

// GetAddress gets wallet address
func (w Wallet) GetAddress() string {
	publicKeyHash := HashPublicKey(w.PublicKey)

	return base58.CheckEncode(publicKeyHash, version)
}

// CreateWallet adds a Wallet into Wallets
func (ws *Wallets) CreateWallet() string {
	wallet := NewWallet()
	address := wallet.GetAddress()

	ws.Wallets[address] = wallet

	return address
}

// NewWallets creates wallets and files it from a file iff it exists
func NewWallets() (*Wallets, error) {
	wallets := Wallets{}
	wallets.Wallets = make(map[string]*Wallet)

	if _, err := os.Stat(walletFile); os.IsNotExist(err) {
		return &wallets, err
	}

	fileContent, err := os.ReadFile(walletFile)
	if err != nil {
		log.Panic(err)
	}

	err = json.Unmarshal(fileContent, &wallets)
	if err != nil {
		log.Panic(err)
	}

	return &wallets, err
}

// SaveToFile saves Wallets into a file
func (ws Wallets) SaveToFile() {
	jsonData, err := json.Marshal(ws)
	if err != nil {
		log.Panic(err)
	}

	err = os.WriteFile(walletFile, jsonData, 0666)
	if err != nil {
		log.Panic(err)
	}
}

// GetAddresses returns addresses stored at wallet file
func (ws *Wallets) GetAddresses() []string {
	var addrs []string

	for address := range ws.Wallets {
		addrs = append(addrs, address)
	}

	return addrs
}

// GetWallet returns a Wallet by address
func (ws Wallets) GetWallet(address string) Wallet {
	return *ws.Wallets[address]
}

func (w Wallet) MarshalJSON() ([]byte, error) {
	mapStringAny := map[string]any{
		"PrivateKey": map[string]any{
			"D": w.PrivateKey.D,
			"PublicKey": map[string]any{
				"X": w.PrivateKey.PublicKey.X,
				"Y": w.PrivateKey.PublicKey.Y,
			},
			"X": w.PrivateKey.X,
			"Y": w.PrivateKey.Y,
		},
		"PublicKey": w.PublicKey,
	}
	return json.Marshal(mapStringAny)
}
