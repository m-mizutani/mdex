package main

import (
	"context"
	"log"
	"os"

	"github.com/m-mizutani/mdex/pkg/cli"
)

func main() {
	if err := cli.New().Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
