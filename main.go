package main

import (
	"blockchain/cli"
	"blockchain/core"
	"github.com/boltdb/bolt"
)

func main() {
	chain := core.GetBlockchain()
	defer func(Db *bolt.DB) {
		err := Db.Close()
		if err != nil {

		}
	}(chain.Db)

	commandLine := cli.Cli{Bc: chain}
	commandLine.Active()
}
