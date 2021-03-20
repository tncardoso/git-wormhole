package main

import (
	"encoding/hex"
	"flag"
	"fmt"

	wormhole "github.com/tncardoso/git-wormhole"
)

func main() {
	strategy := flag.String("strategy", "comment", "[comment]")
	rawPrefix := flag.String("prefix", "", "git commit hash prefix")
	flag.Parse()

	var err error
	var prefix []byte = nil
	if *rawPrefix != "" {
		prefix, err = hex.DecodeString(*rawPrefix)
		if err != nil {
			fmt.Printf("ERROR: prefix is not a valid hex string. Ex. deadc0de")
			return
		}
	}

	wormhole.Collide(*strategy, prefix)
}
