package wormhole

import (
	"bytes"
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

func commentCollisionStrategy(inputChan chan Commit,
	doneChan chan bool, base Commit) {
	defer close(inputChan)
	for tests := 0; tests < 10000000000; tests++ {
		select {
		case done := <-doneChan:
			fmt.Printf("prematurely stopping producer! %#v\n", done)
			return
		default:
			commit := base
			commit.TestId = tests
			commit.Comment = fmt.Sprintf("comment - %d", tests)
			inputChan <- commit
		}
	}
}

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
			fmt.Printf("%d %x %x\n",
				commit.TestId,
				targetPrefix, compHash[:len(targetPrefix)])
		}
		/*if bytes.Equal(targetPrefix[0:3], compHash[:len(targetPrefix)][0:3]) {
		    fmt.Printf("SIMULATED FOUND!!!!\n")
		    fmt.Printf("%d %x %x\n",
		                commit.TestId,
		                targetPrefix, compHash[:len(targetPrefix)])
		    doneChan <- true
		    resultChan <- commit
		    break
		}*/
		if bytes.Equal(targetPrefix, compHash[:len(targetPrefix)]) {
			fmt.Printf("comm %d %x %x %s\n", commit.TestId, targetPrefix, compHash[:len(targetPrefix)], commit.Comment)
			fmt.Printf("FOUND!!!!\n")
			fmt.Printf("%s\n", compHash.String())
			doneChan <- true
			resultChan <- commit
			break
		}
	}
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

func (target *Target) Brute(repo *git.Repository) (*Commit, error) {
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

	overrides := make(map[string]plumbing.Hash)

	var content bytes.Buffer
	targetPath := path.Join(target.path...)
	target.template.Execute(&content, 0)
	overrides[targetPath] = plumbing.ComputeHash(plumbing.BlobObject, content.Bytes())

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
	targetPrefix := []byte{0xde, 0xad}
	now := time.Now()

	/*
		now := time.Now()
		//maxDelta := int64(8*60*60) // 8 hours
		maxDelta := int64(10) // 8 hours
		//targetPrefix := []byte{0xde, 0xad, 0xc0, 0xde}
		tests := 0


			for authorDelta := int64(0); authorDelta < maxDelta; authorDelta++ {
				for commitDelta := authorDelta; commitDelta >= 0; commitDelta-- {
					authorTime := now.Add(-time.Second * time.Duration(authorDelta))
					commitTime := now.Add(time.Second * time.Duration(commitDelta))
					compHash := CommitHash(cfg, treeHash, parentHash, authorTime, commitTime, "comment")
					if bytes.Equal(targetPrefix[0:3], compHash[:len(targetPrefix)][0:3]) {
						fmt.Printf("%d %x %x\n", tests, targetPrefix, compHash[:len(targetPrefix)])

					}
					if tests%1000000 == 0 {
						fmt.Printf("%d %x %x\n", tests, targetPrefix, compHash[:len(targetPrefix)])
					}
					if bytes.Equal(targetPrefix, compHash[:len(targetPrefix)]) {
						fmt.Printf("%d %x %x\n", tests, targetPrefix, compHash[:len(targetPrefix)])
						fmt.Printf("FOUND!!!!\n")
						fmt.Printf("%s\n", compHash.String())
						panic("FOUND")
					}
					tests += 1
				}
			}*/

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

	baseCommit := Commit{
		Config:     cfg,
		TreeHash:   treeHash,
		ParentHash: head.Hash(),
		AuthorTime: now,
		CommitTime: now,
		Comment:    "",
	}

	go commentCollisionStrategy(inputChan, doneChan, baseCommit)
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
