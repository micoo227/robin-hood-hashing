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
	totalPsl    uint64
	maxPsl      uint
	maxFreq     uint
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

	load := float32(float64(m.numElements) / float64(m.size))

	if load >= m.loadFactor {
		m.rehashTable()
	}

	_, ok, i := m.GetWithIndex(key)
	if ok {
		m.elements[i].value = value
		return
	}

	m.insertKeyValuePair(key, value)
}

func (m *Map[K, V]) Get(key K) (V, bool) {
	val, ok, _ := m.GetWithIndex(key)
	return val, ok
}

func (m *Map[K, V]) GetWithIndex(key K) (V, bool, uint64) {
	var zeroVal V
	if m.numElements == 0 {
		return zeroVal, false, 0
	}

	// The PSL of keys clusters around the mean PSL (roughly).
	// Therefore, start search using the mean PSL and iteratively
	// branch out above and below that value.
	downPsl := int(m.totalPsl / m.numElements)
	upPsl := uint(downPsl + 1)

	for ; downPsl >= 0 && upPsl <= m.maxPsl; downPsl, upPsl = downPsl-1, upPsl+1 {
		downIndex := m.getIndexOfKeyAtPsl(key, uint(downPsl))
		upIndex := m.getIndexOfKeyAtPsl(key, upPsl)

		if m.elements[downIndex].set && m.elements[downIndex].key == key {
			return m.elements[downIndex].value, true, downIndex
		}
		if m.elements[upIndex].set && m.elements[upIndex].key == key {
			return m.elements[upIndex].value, true, upIndex
		}
	}

	for ; downPsl >= 0; downPsl-- {
		downIndex := m.getIndexOfKeyAtPsl(key, uint(downPsl))

		if m.elements[downIndex].set && m.elements[downIndex].key == key {
			return m.elements[downIndex].value, true, downIndex
		}
	}

	for ; upPsl <= m.maxPsl; upPsl++ {
		upIndex := m.getIndexOfKeyAtPsl(key, upPsl)

		if m.elements[upIndex].set && m.elements[upIndex].key == key {
			return m.elements[upIndex].value, true, upIndex
		}
	}

	return zeroVal, false, 0
}

func (m *Map[K, V]) Delete(key K) {
	if m.numElements == 0 {
		return
	}

	_, ok, i := m.GetWithIndex(key)

	if ok {
		m.totalPsl -= uint64(m.elements[i].psl)
		m.numElements--
		if m.numElements == 0 {
			m.maxFreq = 0
			m.maxPsl = 0
		} else if m.elements[i].psl == m.maxPsl {
			m.updateMaxStatsOnDelete()
		}
		m.elements[i] = element[K, V]{}

		// Calculate i, j in this way to wrap around array when i, j >= m.size
		for j := (i + 1) % m.size; m.elements[j].set && m.elements[j].psl > 0; i, j = (i+1)%m.size, (j+1)%m.size {
			if m.elements[i].psl == m.maxPsl {
				m.updateMaxStatsOnDelete()
			}
			m.elements[j].psl--
			m.totalPsl--
			m.elements[i] = m.elements[j]
			m.elements[j] = element[K, V]{}
		}
	}
}

func (m *Map[K, V]) getIndexOfKeyAtPsl(key K, psl uint) uint64 {
	encodedBytes := encodeKey(key)
	hash := m.hasher(m.k0, m.k1, encodedBytes)
	i := hash % m.size
	return (i + uint64(psl)) % m.size
}

func (m *Map[K, V]) rehashTable() {
	m.size *= 2
	oldElems := m.elements
	m.elements = make([]element[K, V], m.size)
	m.numElements = 0
	m.totalPsl = 0
	m.maxPsl = 0
	m.maxFreq = 0

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

			m.updateMaxStatsOnInsert(newElem.psl)
			m.totalPsl += uint64(newElem.psl - oldElem.psl)

			newElem = oldElem
		}
		newElem.psl += 1
	}

	m.elements[i] = newElem
	m.numElements++

	m.updateMaxStatsOnInsert(newElem.psl)
	m.totalPsl += uint64(newElem.psl)
}

func (m *Map[K, V]) updateMaxStatsOnInsert(newElemPsl uint) {
	if newElemPsl > m.maxPsl {
		m.maxPsl = newElemPsl
		m.maxFreq = 1
	} else if newElemPsl == m.maxPsl {
		m.maxFreq++
	}
}

func (m *Map[K, V]) updateMaxStatsOnDelete() {
	if m.maxFreq == 1 {
		m.maxPsl--
	} else {
		m.maxFreq--
	}
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
