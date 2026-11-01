package main

import (
	"context"
	"fmt"
	"os"

	"github.com/midsonlajeanty/pulumi-umami/provider"
)

var version = "0.0.1"

func main() {
	built, err := provider.New()
	if err == nil {
		err = built.Run(context.Background(), provider.Name, version)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
