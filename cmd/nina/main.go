package main

import (
	"flag"
	"fmt"
)

var version = "0.1.0"

func main() {
	showVersion := flag.Bool("version", false, "Show nina version")
	flag.Parse()

	if *showVersion {
		fmt.Println("nina", version)
		return
	}

	fmt.Println("Nina")
}
