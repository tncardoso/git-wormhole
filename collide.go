package wormhole

import (
	"fmt"

	"github.com/go-git/go-git/v5"
)

func Collide() {
	repo, err := git.PlainOpen(".")
	if err != nil {
		panic(err)
	}

	target, err := New("inside/version.py", "version.template")
	if err != nil {
		panic(err)
	}

	result, err := target.Brute(repo)
	if err != nil {
		panic(err)
	}

	if result == nil {
		fmt.Printf("Could not find hash collision.\n")
	} else {
		fmt.Printf("Found collision.\n")
		fmt.Printf("%s", string(result.Bytes()))
		fmt.Printf("GIT_AUTHOR_DATE=\"raw:%d %s\" ", result.AuthorTime.Unix(), result.AuthorTime.Format("-0700"))
		fmt.Printf("GIT_COMMITTER_DATE=\"raw:%d %s\" ", result.CommitTime.Unix(), result.CommitTime.Format("-0700"))
		fmt.Printf("git commit -a -m '%s'\n", result.Comment)
	}
}
