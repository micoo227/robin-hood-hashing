package rhmap

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"log"

	"github.com/dchest/siphash"
)

// Item in hashmap
type element[K comparable, V any] struct {
	key   K
	value V
	psl   int
	set   bool
}

// Implementation of robin hood hashmap
type Map[K comparable, V any] struct {
	hasher      func(k0, k1 uint64, p []byte) uint64
	numElements uint64
	elements    []element[K, V]
	size        uint64
	loadFactor  float32
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
			temp := m.elements[i]
			m.elements[i] = newElem
			newElem = temp
		}
		newElem.psl += 1
	}

	m.elements[i] = newElem
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
	var k [16]byte
	for i := range k {
		k[i] = byte(i)
	}

	k0 := binary.LittleEndian.Uint64(k[0:8])
	k1 := binary.LittleEndian.Uint64(k[8:16])
	return k0, k1
}
