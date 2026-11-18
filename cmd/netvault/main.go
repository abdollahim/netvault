// Command netvault archives device command output and tracks changes.
package main

import (
	"fmt"
	"os"
)

// version is set at build time: go build -ldflags "-X main.version=0.1.0".
var version = "0.0.0-dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Println("netvault", version)
		return
	}
	fmt.Fprintln(os.Stderr, "netvault: no commands yet, see README")
	os.Exit(2)
}