package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"

	"github.com/go-git/go-git/v5"
	wormhole "github.com/tncardoso/git-wormhole"
)

func main() {
	rawStrategy := flag.String("strategy", "time", "[comment|time]")
	rawPrefix := flag.String("prefix", "", "git commit hash prefix")
	comment := flag.String("comment", "git-wormhole", "comment that should be used on commit")
	maxTests := flag.Int("tests", 0x80000000, "maximum number of tests in comment strategy")
	maxDelta := flag.Int64("delta", 6*60*60, "max time change in author date in seconds (default: 6 hours)")
	maxDiff := flag.Int64("diff", 60*60, "max difference between author and commit dates (default: 1 hour)")
	flag.Parse()

	args := flag.Args()
	if len(args) != 2 {
		fmt.Printf("ERROR: missing template (in golang template format) and target file.\n")
		fmt.Printf("usage: %s [TEMPLATE] [TARGET]\n", os.Args[0])
		fmt.Printf("Template format: https://golang.org/pkg/text/template/\n")
		return
	}

	var err error
	var prefix []byte = nil
	if *rawPrefix != "" {
		prefix, err = hex.DecodeString(*rawPrefix)
		if err != nil {
			fmt.Printf("ERROR: prefix is not a valid hex string. Ex. deadc0de\n")
			return
		}
	}

	var strategy wormhole.Strategy
	if *rawStrategy == "comment" {
		fmt.Printf("strategy > comment max_tests= %d\n", *maxTests)
		strategy = wormhole.NewCommentStrategy(*maxTests)
	} else if *rawStrategy == "time" {
		strategy = wormhole.NewTimeStrategy(*maxDelta, *maxDiff)
	} else {
		fmt.Printf("ERROR: invalid strategy. Options: comment, date\n")
		fmt.Printf("    comment: add suffix to commit comment.\n")
		fmt.Printf("    date: jitter author and commit dates.\n")
		return
	}

	repo, err := git.PlainOpen(".")
	if err != nil {
		panic(err)
	}

	fmt.Printf("template > %s\n", args[0])
	fmt.Printf("target > %s\n", args[1])
	target, err := wormhole.New(args[1], args[0])
	if err != nil {
		panic(err)
	}

	result, err := target.Brute(repo, strategy, prefix, *comment)
	if err != nil {
		panic(err)
	}

	if result == nil {
		fmt.Printf("Could not find hash collision.\n")
	} else {
		fmt.Printf("\nFound collision.\nRun the following command:\n\n")
		//fmt.Printf("%s", string(result.Bytes()))
		fmt.Printf("GIT_AUTHOR_DATE=\"raw:%d %s\" ", result.AuthorTime.Unix(), result.AuthorTime.Format("-0700"))
		fmt.Printf("GIT_COMMITTER_DATE=\"raw:%d %s\" ", result.CommitTime.Unix(), result.CommitTime.Format("-0700"))
		fmt.Printf("git commit -a -m '%s'\n", result.Comment)
	}
}
