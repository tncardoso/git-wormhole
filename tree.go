package wormhole

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"path"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TreeHash(tree *object.Tree, prefix string, overrides map[string]plumbing.Hash) plumbing.Hash {
	var content bytes.Buffer
	for _, entry := range tree.Entries {
		entryPath := path.Join(prefix, entry.Name)
		newHash, inOverride := overrides[entryPath]

		content.Write([]byte(strings.TrimLeftFunc(entry.Mode.String(), func(r rune) bool {
			return r == '0'
		})))
		content.Write([]byte(" "))
		content.Write([]byte(entry.Name))
		content.Write([]byte("\x00"))
		if inOverride {
			content.Write(newHash[:])
		} else {
			content.Write(entry.Hash[:])
		}
	}

	final := []byte(fmt.Sprintf("tree %d\x00", content.Len()))
	final = append(final, content.Bytes()...)
	hash := sha1.New()
	hash.Write(final)
	digest := hash.Sum(nil)
	var finalDigest [20]byte
	copy(finalDigest[:], digest)
	return plumbing.Hash(finalDigest)
}
