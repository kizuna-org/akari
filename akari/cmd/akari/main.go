package main

import (
	"flag"
	"log/slog"
)

var version = "0.1.0"

func main() {
	showVersion := flag.Bool("version", false, "Show akari version")
	flag.Parse()

	if *showVersion {
		slog.Info("akari version", "version", version)

		return
	}

	slog.Info("Akari")
}
