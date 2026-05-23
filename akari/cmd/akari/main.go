package main

import "github.com/kizuna-org/akari/internal/app"

//nolint:gochecknoglobals
var run = func() {
	app.New().Run()
}

func main() {
	run()
}
