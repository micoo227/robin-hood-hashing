package rhmap

import (
	"bytes"
	"encoding/gob"
	"log"
	"math/rand"

	"github.com/dchest/siphash"
)

// Default size for hash map when no size is specified on instantiation
const defaultSize uint64 = 8

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
	k0          uint64
	k1          uint64
	numElements uint64
	elements    []element[K, V]
	size        uint64
	loadFactor  float32
	averagePsl  uint
	maxPsl      uint
	minPsl      uint
}

func New[K comparable, V any](size ...uint64) *Map[K, V] {
	mapSize := defaultSize
	if len(size) > 0 && size[0] > 0 {
		mapSize = size[0]
	}

	return &Map[K, V]{
		hasher:      siphash.Hash,
		k0:          rand.Uint64(),
		k1:          rand.Uint64(),
		numElements: 0,
		elements:    make([]element[K, V], mapSize),
		size:        mapSize,
		loadFactor:  .9,
	}
}

func (m *Map[K, V]) Set(key K, value V) {

	load := m.numElements / m.size

	if load >= uint64(m.loadFactor) {
		m.rehashTable()
	}

	m.insertKeyValuePair(key, value)
}

func (m *Map[K, V]) Get(key K) (V, bool) {
	val, ok, _ := m.GetWithIndex(key)
	return val, ok
}

func (m *Map[K, V]) GetWithIndex(key K) (V, bool, uint64) {
	// The PSL of keys clusters around the mean PSL (roughly).
	// Therefore, start search using the mean PSL and iteratively
	// branch out above and below that value.
	downPsl := m.averagePsl
	upPsl := downPsl + 1

	for ; downPsl >= m.minPsl && upPsl <= m.maxPsl; downPsl, upPsl = downPsl-1, upPsl+1 {
		downIndex := m.getIndexOfKeyAtPsl(key, downPsl)
		if m.elements[downIndex].set && m.elements[downIndex].key == key {
			return m.elements[downIndex].value, true, downIndex
		}

		upIndex := m.getIndexOfKeyAtPsl(key, upPsl)
		if m.elements[upIndex].set && m.elements[upIndex].key == key {
			return m.elements[upIndex].value, true, downIndex
		}
	}

	var zeroVal V
	return zeroVal, false, 0
}

func (m *Map[K, V]) Delete(key K) {
	_, ok, i := m.GetWithIndex(key)

	if ok {
		m.elements[i] = element[K, V]{}

		for elem := m.elements[i+1]; elem.set && elem.psl > 0; elem = m.elements[i+1] {
			elem.psl--
			m.elements[i] = elem
			m.elements[i+1] = element[K, V]{}
			i++
		}
	}
}

func (m *Map[K, V]) getIndexOfKeyAtPsl(key K, psl uint) uint64 {
	encodedBytes := encodeKey(key)
	hash := m.hasher(m.k0, m.k1, encodedBytes)
	i := hash % m.size
	return i + uint64(psl)
}

func (m *Map[K, V]) rehashTable() {
	m.size *= 2
	oldElems := m.elements
	m.elements = make([]element[K, V], m.size)

	for _, elem := range oldElems {
		m.insertKeyValuePair(elem.key, elem.value)
	}
}

func (m *Map[K, V]) insertKeyValuePair(key K, value V) {
	encodedBytes := encodeKey(key)
	hash := m.hasher(m.k0, m.k1, encodedBytes)
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

func (m *Map[K, V]) Len() uint64 {
	return m.numElements
}
