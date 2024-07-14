package main

import (
	"blockchain/core"
)

func main() {
	chain := core.GetBlockchain()
	chain.AddBlock("Genesis Block")
	chain.AddBlock("Second Block")
	chain.AddBlock("Third Block")
	chain.ShowBlocks()
}
