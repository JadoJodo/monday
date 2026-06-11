// Command rundown automates routine macOS maintenance.
package main

import (
	"context"
	"os"

	"github.com/JadoJodo/rundown/cmd"
)

func main() {
	if err := cmd.Execute(context.Background()); err != nil {
		os.Exit(1)
	}
}
