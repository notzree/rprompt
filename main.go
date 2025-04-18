package main

import (
	"context"
	"log"
	"os"

	"github.com/notzree/rprompt/v2/prompt"
)

func main() {
	cmd := prompt.InitCLI()
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
