package once

import (
	"sync"
)

type Once[K comparable, V ~bool] struct {
	store map[K]V
	rw    *sync.RWMutex
}

func New[K comparable, V ~bool]() *Once[K, V] {
	return &Once[K, V]{
		store: make(map[K]V),
		rw:    new(sync.RWMutex),
	}
}

func (o *Once[K, V]) Stored(key K) V {
	o.rw.Lock()
	defer o.rw.Unlock()
	v, ok := o.store[key]
	if !ok {
		o.store[key] = true
		return false
	}
	return v
}
