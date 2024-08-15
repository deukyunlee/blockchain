package main

import (
	"blockchain/cli"
	"blockchain/core"
)

func main() {
	chain := core.GetBlockchain()
	defer chain.Db.Close()

	commandLine := cli.Cli{chain}
	commandLine.Active()
}
