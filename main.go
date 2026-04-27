// Package main provides the entry point for the Seton application.
package main

import (
	"log"
	"os"

	"github.com/hdirksor/seton/commands"
)

func main() {

	cli := commands.InitRootCmd()
	err := cli.Execute()

	if err != nil {
		log.Printf("Error: %v", err)
		os.Exit(1)
	}
}
