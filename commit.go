package wormhole

import (
	"bytes"
	"fmt"
	"time"

	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
)

type Commit struct {
	TestId     int
	Config     *config.Config
	TreeHash   plumbing.Hash
	ParentHash plumbing.Hash
	AuthorTime time.Time
	CommitTime time.Time
	Comment    string
}

func (commit Commit) Bytes() []byte {
	var buff bytes.Buffer
	buff.Write([]byte("tree "))
	buff.Write([]byte(commit.TreeHash.String()))
	buff.Write([]byte("\nparent "))
	buff.Write([]byte(commit.ParentHash.String()))
	buff.Write([]byte("\nauthor "))
	buff.Write([]byte(commit.Config.User.Name))
	buff.Write([]byte(" <"))
	buff.Write([]byte(commit.Config.User.Email))
	buff.Write([]byte("> "))
	buff.Write([]byte(fmt.Sprintf("%d ", commit.AuthorTime.Unix())))
	buff.Write([]byte(commit.AuthorTime.Format("-0700")))
	buff.Write([]byte("\ncommitter "))
	buff.Write([]byte(commit.Config.User.Name))
	buff.Write([]byte(" <"))
	buff.Write([]byte(commit.Config.User.Email))
	buff.Write([]byte("> "))
	buff.Write([]byte(fmt.Sprintf("%d ", commit.CommitTime.Unix())))
	buff.Write([]byte(commit.CommitTime.Format("-0700")))
	buff.Write([]byte("\n\n"))
	buff.Write([]byte(commit.Comment))
	buff.Write([]byte("\n"))
	return buff.Bytes()
}

func (commit Commit) Hash() plumbing.Hash {
	return plumbing.ComputeHash(plumbing.CommitObject, commit.Bytes())
}
