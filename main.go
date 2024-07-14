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
	}
}
