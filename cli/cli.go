package cli

import (
	"blockchain/core"
	"flag"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/btcsuite/btcutil/base58"
	"log"
	"os"
	"strconv"
)

type Cli struct {
	Bc *core.Blockchain
}

func (cli *Cli) Active() {
	if len(os.Args) < 2 {
		cli.printUsage()
		os.Exit(1)
	}
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	createBlockchainCmd := flag.NewFlagSet("createblockchain", flag.ExitOnError)
	showBlocksCmd := flag.NewFlagSet("showblocks", flag.ExitOnError)
	getBalanceCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
	createWalletCmd := flag.NewFlagSet("createwallet", flag.ExitOnError)
	showAddrsCmd := flag.NewFlagSet("showaddresses", flag.ExitOnError)

	sendFrom := sendCmd.String("from", "", "Source address")
	sendTo := sendCmd.String("to", "", "Destination address")
	sendAmount := sendCmd.Int("amount", 0, "Amount to send")
	createBlockchainAddr := createBlockchainCmd.String("address", "", "First Miner's address")
	getBalanceAddress := getBalanceCmd.String("address", "", "The address to get balance for")

	switch os.Args[1] {
	case "send":
		err := sendCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "createblockchain":
		err := createBlockchainCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "showblocks":
		err := showBlocksCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "getbalance":
		err := getBalanceCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "createwallet":
		err := createWalletCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "showaddresses":
		err := showAddrsCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	default:
		cli.printUsage()
		os.Exit(1)
	}

	if sendCmd.Parsed() {
		if *sendFrom == "" || *sendTo == "" || *sendAmount <= 0 {
			sendCmd.Usage()
			os.Exit(1)
		}
		cli.send(*sendFrom, *sendTo, *sendAmount)
	}

	if createBlockchainCmd.Parsed() {
		if *createBlockchainAddr == "" {
			createBlockchainCmd.Usage()
			os.Exit(1)
		}
		cli.createBlockchain(*createBlockchainAddr)
	}

	if showBlocksCmd.Parsed() {
		cli.showBlocks()
	}

	if getBalanceCmd.Parsed() {
		if *getBalanceAddress == "" {
			getBalanceCmd.Usage()
			os.Exit(1)
		}
		cli.getBalance(*getBalanceAddress)
	}

	if createWalletCmd.Parsed() {
		cli.createWallet()
	}

	if showAddrsCmd.Parsed() {
		cli.showAddresses()
	}
}

func (cli *Cli) send(from, to string, amount int) {
	bc := core.GetBlockchain()
	defer func(Db *bolt.DB) {
		err := Db.Close()
		if err != nil {

		}
	}(bc.Db)
	tx := core.NewUTXOTransaction(from, to, amount, bc)
	rwTx := core.NewCoinbaseTX(from, "Mining reward")
	bc.AddBlock([]*core.Transaction{rwTx, tx})
	fmt.Println("Send Complete!!")
}

func (cli *Cli) createBlockchain(address string) {
	newBc := core.CreateBlockchain(address)
	newBc.Db.Close()
	fmt.Println("Successfully done with create blockchain!")
}

// Show Blockchains
func (cli *Cli) showBlocks() {
	bc := core.GetBlockchain()
	defer bc.Db.Close()
	bcI := bc.Iterator()
	for {
		block := bcI.GetNextBlock()
		pow := core.NewProofOfWork(block)

		fmt.Println("\nTimeStamp:", block.TimeStamp)
		for index := range block.Transactions {
			fmt.Println("Transactions: ")
			fmt.Printf(" ID: %v\n", block.Transactions[index].ID)
			fmt.Printf(" Vin: %v\n", block.Transactions[index].Vin[0])
			fmt.Printf("    .ScriptSig: %v\n", block.Transactions[index].Vin[0].ScriptSig)
			fmt.Printf(" Vout: %v\n", block.Transactions[index].Vout)
		}
		fmt.Printf("Hash: %x\n", block.Hash)
		fmt.Printf("Prev Hash: %x\n", block.PrevHash)
		fmt.Printf("Nonce: %d\n", block.Nonce)
		fmt.Printf("is Validated: %s\n", strconv.FormatBool(pow.Validate()))

		if len(block.PrevHash) == 0 {
			break
		}
	}
}

func (cli *Cli) getBalance(address string) {
	bc := core.GetBlockchain()
	defer bc.Db.Close()

	balance := 0

	publicKeyHash, _, err := base58.CheckDecode(address)
	if err != nil {
		log.Panic(err)
	}
	utxs := bc.FindUnspentTxs(publicKeyHash)

	for _, tx := range utxs {
		for _, out := range tx.Vout {
			if out.IsLockedWithKey(publicKeyHash) {
				balance += out.Value
			}
		}
	}

	fmt.Printf("Balance of '%s': %d\n", address, balance)
}

func (cli *Cli) createWallet() {
	wallets, _ := core.NewWallets()
	address := wallets.CreateWallet()
	wallets.SaveToFile()

	fmt.Printf("Your new address: %s\n", address)
}

func (cli *Cli) showAddresses() {
	wallets, err := core.NewWallets()
	if err != nil {
		log.Panic(err)
	}
	addresses := wallets.GetAddresses()

	for _, address := range addresses {
		fmt.Println(address)
	}
}

func (cli *Cli) printUsage() {
	fmt.Printf("How to use:\n\n")
	fmt.Println("  send -from FROM -to TO -amount AMOUNT - send AMOUNT of coins from FROM address to TO")
	fmt.Println("  createblockchain -address ADDRESS - create new blockchain")
	fmt.Println("  showblocks - print all the blocks of the blockchain")
	fmt.Println("  getbalance -address ADDRESS - Get balance of ADDRESS")
	fmt.Println("  createwallet - Create your Wallet")
	fmt.Println("  showaddresses - Show all addresses")
}
