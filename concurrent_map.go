package main

import "sync"

type (
	key   interface{}
	value interface{}
)

// ConcurrentMap implements a generic thread-safe (I think) map.
type ConcurrentMap interface {
	SetIfAbsent(key, value) bool
	Set(key, value)
	Remove(key)
	Map() map[key]value
}

type concurrentMap struct {
	mapping map[key]value
	mutex   sync.RWMutex
}

func NewConcurrentMap() ConcurrentMap {
	concurrentMap := concurrentMap{
		mapping: make(map[key]value),
		mutex:   sync.RWMutex{},
	}

	return &concurrentMap
}

func (c *concurrentMap) SetIfAbsent(k key, v value) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if _, ok := c.mapping[k]; ok {
		return false
	}

	c.mapping[k] = v

	return true
}

func (c *concurrentMap) Set(k key, v value) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.mapping[k] = v
}

func (c *concurrentMap) Remove(k key) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.mapping, k)
}

func (c *concurrentMap) Map() map[key]value {
	return c.mapping
}
