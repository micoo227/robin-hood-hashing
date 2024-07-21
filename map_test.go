package rhmap

import (
	"strconv"
	"testing"
)

func TestMapCreation(t *testing.T) {
	m := New[int, int]()
	if m.Len() != 0 {
		t.Errorf("New map should be empty but has %d items.", m.Len())
	}
	// TODO: finish
}

func TestSet(t *testing.T) {
	m := New[int, string]()

	for i := 1; i <= 10; i++ {
		m.Set(i, strconv.Itoa(i))
	}

	for i := 1; i <= 10; i++ {
		val, ok := m.Get(i)
		if !ok {
			t.Errorf("Ok should be true for key %d stored in the map.", i)
			continue
		}
		if val != strconv.Itoa(i) {
			t.Errorf("Val mapped to key %d was %s. Expected %s", i, val, strconv.Itoa(i))
		}
	}
}
