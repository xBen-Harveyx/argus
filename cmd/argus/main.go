package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/ben/argus/internal/config"
	"github.com/ben/argus/internal/run"
)

func main() {
	cfg, err := config.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	if err := run.Execute(context.Background(), cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		if errors.Is(err, run.ErrPartialFailure) {
			os.Exit(1)
		}
		os.Exit(1)
	}
}
