package main

import (
	"log"

	"github.com/matope/ratchet/command"
)

func main() {
	log.SetFlags(0)
	command.RootCmd().Execute()
}
