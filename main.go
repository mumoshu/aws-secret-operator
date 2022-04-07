package main

import (
	"os"

	"github.com/mumoshu/aws-secret-operator/cmd"
)

func main() {
	if err := cmd.Root.Execute(); err != nil {
		os.Exit(1)
	}
}
