package controllers

import (
	"context"
	"strings"
	"sync"

	"github.com/milvus-io/milvus-operator/pkg/config"
	"github.com/pkg/errors"
)

type Group struct {
	wait   sync.WaitGroup
	cancel context.CancelFunc

	// Collect all the errors
	errors []error
	locker *sync.Mutex
}

// NewGroup creates a new group.
func NewGroup(ctx context.Context) (*Group, context.Context) {
	gtx, cancel := context.WithCancel(ctx)

	return &Group{
		cancel: cancel,
		errors: make([]error, 0),
		locker: &sync.Mutex{},
	}, gtx
}

// Go to run a func.
func (g *Group) Go(f func() error) {
	g.wait.Add(1)

	go func() {
		if !config.IsDebug() {
			defer func() {
				if err := recover(); err != nil {
					g.locker.Lock()
					g.errors = append(g.errors, err.(error))
					g.locker.Unlock()
				}
			}()
		}
		defer g.wait.Done()

		// Run and handle error
		if err := f(); err != nil {
			g.locker.Lock()
			defer g.locker.Unlock()

			g.errors = append(g.errors, err)
		}
	}()
}

// Wait until all the go functions are returned
// If errors occurred, they'll be combined with ":" and returned.
func (g *Group) Wait() error {
	g.wait.Wait()

	defer func() {
		if g.cancel != nil {
			// Send signals to the downstream components
			g.cancel()
		}
	}()

	errTexts := make([]string, 0)
	for _, e := range g.errors {
		errTexts = append(errTexts, e.Error())
	}

	if len(errTexts) > 0 {
		return errors.Errorf("groups error: %s", strings.Join(errTexts, ":"))
	}

	return nil
}
