package main

import (
	"blockchain/core"
	"time"
)

func main() {
	chain := core.GetBlockchain()
	for {
		chain.AddBlock("New Block")
		time.Sleep(1 * time.Second)

		if len(chain.Blocks) > 10 {
			break
		}
	}

	chain.ShowBlocks()
}
