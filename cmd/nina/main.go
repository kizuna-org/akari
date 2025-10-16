package main

import (
	"flag"
	"log/slog"
)

var version = "0.1.0"

func main() {
	showVersion := flag.Bool("version", false, "Show nina version")
	flag.Parse()

	if *showVersion {
		slog.Info("nina version", "version", version)

		return
	}

	slog.Info("Nina")
}
