package wormhole

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"path"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
)

func worker(wg *sync.WaitGroup, inputChan chan Commit, resultChan chan Commit,
	doneChan chan bool, targetPrefix []byte) {
	defer wg.Done()
	for {
		commit, ok := <-inputChan
		if !ok {
			break
		}
		compHash := commit.Hash()

		if commit.TestId%1000000 == 0 {
			fmt.Printf("    test_id= %d target= %x  current= %x\n",
				commit.TestId,
				targetPrefix,
				compHash[:len(targetPrefix)])
		}

		if bytes.Equal(targetPrefix, compHash[:len(targetPrefix)]) {
			doneChan <- true
			resultChan <- commit
			break
		}
	}
}

type TemplateData struct {
	Prefix []byte
}

type Target struct {
	path     []string
	template *template.Template
}

func New(targetPath string, templatePath string) (*Target, error) {
	templateContent, err := ioutil.ReadFile(templatePath)
	if err != nil {
		return nil, err
	}

	temp, err := template.New("templatePath").Parse(string(templateContent))
	if err != nil {
		return nil, err
	}

	return &Target{
		path:     strings.Split(targetPath, "/"),
		template: temp,
	}, nil
}

func (target *Target) Brute(repo *git.Repository, strategy Strategy, targetPrefix []byte, comment string) (*Commit, error) {
	if targetPrefix == nil {
		targetPrefix = make([]byte, 2)
		rand.Read(targetPrefix)
	}
	fmt.Printf("target_prefix > %s\n", hex.EncodeToString(targetPrefix))

	head, err := repo.Head()
	if err != nil {
		panic(err)
	}

	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		panic(err)
	}
	fmt.Printf("parent_commit > %s\n", head.Hash().String())

	tree, err := repo.TreeObject(commit.TreeHash)
	if err != nil {
		panic(err)
	}

	data := &TemplateData{
		Prefix: targetPrefix,
	}

	overrides := make(map[string]plumbing.Hash)
	var content bytes.Buffer
	targetPath := path.Join(target.path...)
	target.template.Execute(&content, data)
	overrides[targetPath] = plumbing.ComputeHash(plumbing.BlobObject, content.Bytes())

	// write target with template
	err = ioutil.WriteFile(targetPath, content.Bytes(), 0644)
	if err != nil {
		return nil, err
	}

	// update directories hashes
	for i := len(target.path) - 1; i > 0; i-- {
		subPath := target.path[0:i]
		subPathStr := path.Join(subPath...)

		entry, err := tree.FindEntry(subPathStr)
		if err != nil {
			return nil, err
		}

		subTree, err := repo.TreeObject(entry.Hash)
		subTreeHash := TreeHash(subTree, subPathStr, overrides)
		overrides[subPathStr] = subTreeHash
	}

	// compute final tree hash
	treeHash := TreeHash(tree, "", overrides)
	fmt.Printf("expected_tree_hash > %s\n", treeHash.String())

	cfg, err := repo.ConfigScoped(config.SystemScope)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Searching for collision...\n")
	inputBuffSize := 4096
	workers := 6
	var wg sync.WaitGroup

	inputChan := make(chan Commit, inputBuffSize)
	resultChan := make(chan Commit, workers)
	doneChan := make(chan bool, workers) // 1 for strategy

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go worker(&wg, inputChan, resultChan, doneChan, targetPrefix)
	}

	now := time.Now()
	baseCommit := Commit{
		Config:     cfg,
		TreeHash:   treeHash,
		ParentHash: head.Hash(),
		AuthorTime: now,
		CommitTime: now,
		Comment:    comment,
	}

	go strategy.Producer(inputChan, doneChan, baseCommit)
	wg.Wait()

	select {
	case result := <-resultChan:
		fmt.Printf("Found value! %s\n", result.Hash().String())
		return &result, nil
	default:
		fmt.Printf("No value found :(\n")
		return nil, nil
	}

}
