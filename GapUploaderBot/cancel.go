package main

import (
	"context"
	"sync"
)

type CancelMap struct {
	m  map[string]context.CancelFunc
	mu sync.RWMutex
}

func NewCancelMap() *CancelMap {
	return &CancelMap{
		m: make(map[string]context.CancelFunc),
	}
}

func (c *CancelMap) Add(id string, cancel context.CancelFunc) {
	c.mu.Lock()
	c.m[id] = cancel
	c.mu.Unlock()
}

func (c *CancelMap) Cancel(id string) {
	c.mu.RLock()
	if cancel, ok := c.m[id]; ok {
		cancel()
	}
	c.mu.RUnlock()
}

func (c *CancelMap) Delete(id string) {
	c.mu.Lock()
	delete(c.m, id)
	c.mu.Unlock()
}
