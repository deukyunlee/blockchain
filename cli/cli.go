package cli

import (
	"blockchain/core"
	"flag"
	"fmt"
	"log"
	"os"
)

type Cli struct {
	Bc *core.Blockchain
}

func (cli *Cli) Active() {
	if len(os.Args) < 2 {
		cli.printUsage()
		os.Exit(1)
	}
	addBlockCmd := flag.NewFlagSet("addblock", flag.ExitOnError)
	showBlocksCmd := flag.NewFlagSet("showblocks", flag.ExitOnError)

	addBlockData := addBlockCmd.String("data", "", "Block data")

	switch os.Args[1] {
	case "addblock":
		err := addBlockCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "showblocks":
		err := showBlocksCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	default:
		cli.printUsage()
		os.Exit(1)
	}

	if addBlockCmd.Parsed() {
		if *addBlockData == "" {
			addBlockCmd.Usage()
			os.Exit(1)
		}
		cli.Bc.AddBlock(*addBlockData)
	}

	if showBlocksCmd.Parsed() {
		cli.Bc.ShowBlocks()
	}
}

func (cli *Cli) printUsage() {
	fmt.Printf("How to use:\n\n")
	fmt.Println("  addblock -data DATA - add a block to the blockchain")
	fmt.Println("  showblocks - print all the blocks of the blockchain")
}
