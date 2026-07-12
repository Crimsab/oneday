package main

import (
	"fmt"
	"os"

	gatewayschema "github.com/crimsab/oneday/internal/game/gatewayprotocol/schema"
)

func main() {
	payload, err := gatewayschema.Generate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "encode gateway schema: %v\n", err)
		os.Exit(1)
	}
	if _, err := os.Stdout.Write(payload); err != nil {
		fmt.Fprintf(os.Stderr, "write gateway schema: %v\n", err)
		os.Exit(1)
	}
}
