package wormhole

import (
	"fmt"
	"time"
)

type Strategy interface {
	Producer(inputChan chan Commit, doneChan chan bool, base Commit)
}

type CommentStrategy struct {
	maxTests int
}

func NewCommentStrategy(maxTests int) *CommentStrategy {
	return &CommentStrategy{maxTests: maxTests}
}

func (cs *CommentStrategy) Producer(inputChan chan Commit, doneChan chan bool, base Commit) {
	defer close(inputChan)
	for tests := 0; tests < cs.maxTests; tests++ {
		select {
		case done := <-doneChan:
			fmt.Printf("prematurely stopping producer! %#v\n", done)
			return
		default:
			commit := base
			commit.TestId = tests
			commit.Comment = fmt.Sprintf("%s\n%d", base.Comment, tests)
			inputChan <- commit
		}
	}
}

type TimeStrategy struct {
	// Maximum seconds of divergence
	maxDelta int64
	// Keep dates within maxDiff seconds
	maxDiff int64
}

func NewTimeStrategy(maxDelta int64, maxDiff int64) *TimeStrategy {
	return &TimeStrategy{
		maxDelta: maxDelta,
		maxDiff:  maxDiff,
	}
}

func (ts *TimeStrategy) Producer(inputChan chan Commit, doneChan chan bool, base Commit) {
	defer close(inputChan)

	tests := 0
	for authorDelta := int64(0); authorDelta < ts.maxDelta; authorDelta++ {
		for commitDelta := int64(0); commitDelta < ts.maxDiff; commitDelta++ {
			select {
			case done := <-doneChan:
				fmt.Printf("prematurely stopping producer! %#v\n", done)
				return
			default:
				authorTime := base.AuthorTime.Add(-time.Second * time.Duration(authorDelta))
				commitTime := authorTime.Add(-time.Second * time.Duration(commitDelta))
				commit := base
				commit.TestId = tests
				commit.AuthorTime = authorTime
				commit.CommitTime = commitTime
				inputChan <- commit
			}
			tests += 1
		}
	}
}
