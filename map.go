package rhmap

import (
	"sync/atomic"

	"golang.org/x/exp/constraints"
)

type (
	hashable interface {
		constraints.Integer | constraints.Float | ~string
	}

	// Item in hashmap
	element[K hashable, V any] struct {
		keyHash uintptr
		key     K
		value   atomic.Pointer[V]
	}

	// Implementation of robin hood hashmap
	Map[K hashable, V any] struct {
		hasher      func(K) uintptr
		numElements atomic.Uintptr
		elements    []element[K, V]
		defaultSize uintptr
	}
)
