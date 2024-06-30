package rhmap

import (
	"bytes"
	"encoding/gob"
	"log"
	"math/rand"

	"github.com/dchest/siphash"
)

// Item in hashmap
type element[K comparable, V any] struct {
	key   K
	value V
	psl   uint
	set   bool
}

// Implementation of robin hood hashmap
type Map[K comparable, V any] struct {
	hasher      func(k0, k1 uint64, p []byte) uint64
	numElements uint64
	elements    []element[K, V]
	size        uint64
	loadFactor  float32
	averagePsl  uint
	maxPsl      uint
	minPsl      uint
}

func New[K comparable, V any](size uint64) *Map[K, V] {
	return &Map[K, V]{
		hasher:      siphash.Hash,
		numElements: 0,
		elements:    make([]element[K, V], size),
		size:        size,
		loadFactor:  .9,
	}
}

func (m *Map[K, V]) Set(key K, value V) {

	load := m.numElements / m.size

	if load >= uint64(m.loadFactor) {
		m.rehashTable()
	}

	k0, k1 := createHashingKeys()
	m.insertKeyValuePair(key, value, k0, k1)
}

func (m *Map[K, V]) Get(key K) (value V, ok bool) {
	// The PSL of keys clusters around the mean PSL (roughly).
	// Therefore, start search using the mean PSL and iteratively
	// branch out above and below that value.
	downPsl := m.averagePsl
	upPsl := downPsl + 1
	k0, k1 := createHashingKeys()

	for ; downPsl >= m.minPsl && upPsl <= m.maxPsl; downPsl, upPsl = downPsl-1, upPsl+1 {
		downIndex := m.getIndexOfKeyAtPsl(key, downPsl, k0, k1)
		if m.elements[downIndex].key == key {
			return m.elements[downIndex].value, true
		}

		upIndex := m.getIndexOfKeyAtPsl(key, upPsl, k0, k1)
		if m.elements[upIndex].key == key {
			return m.elements[upIndex].value, true
		}
	}

	var zeroVal V
	return zeroVal, false
}

func (m *Map[K, V]) getIndexOfKeyAtPsl(key K, psl uint, k0, k1 uint64) uint64 {
	encodedBytes := encodeKey(key)
	hash := m.hasher(k0, k1, encodedBytes)
	i := hash % m.size
	return i + uint64(psl)
}

func (m *Map[K, V]) rehashTable() {
	m.size *= 2
	oldElems := m.elements
	m.elements = make([]element[K, V], m.size)

	k0, k1 := createHashingKeys()

	for _, elem := range oldElems {
		m.insertKeyValuePair(elem.key, elem.value, k0, k1)
	}
}

func (m *Map[K, V]) insertKeyValuePair(key K, value V, k0, k1 uint64) {
	encodedBytes := encodeKey(key)
	hash := m.hasher(k0, k1, encodedBytes)
	i := hash % m.size

	newElem := element[K, V]{key: key, value: value, psl: 0, set: true}
	// Calculate i in this way to wrap around array when i >= m.size
	for ; m.elements[i].set; i = (i + 1) % m.size {
		if newElem.psl > m.elements[i].psl {
			oldElem := m.elements[i]
			m.elements[i] = newElem

			m.updatePslAverage(oldElem.psl, newElem.psl)

			newElem = oldElem
		}
		newElem.psl += 1
	}

	m.elements[i] = newElem
	m.numElements++

	if newElem.psl > m.maxPsl {
		m.maxPsl = newElem.psl
	}
	m.averagePsl = uint((uint64(m.averagePsl)*m.numElements + uint64(newElem.psl)) / m.numElements)
}

func (m *Map[K, V]) updatePslAverage(oldPsl, newPsl uint) {
	m.averagePsl = uint((uint64(m.averagePsl)*m.numElements - uint64(oldPsl) + uint64(newPsl)) / m.numElements)
}

func encodeKey[T comparable](key T) []byte {
	var buffer bytes.Buffer
	enc := gob.NewEncoder(&buffer)
	err := enc.Encode(key)
	if err != nil {
		log.Fatal("Could not encode key: ", err)
	}
	return buffer.Bytes()
}

func createHashingKeys() (uint64, uint64) {
	k0, k1 := rand.Uint64(), rand.Uint64()
	return k0, k1
}
