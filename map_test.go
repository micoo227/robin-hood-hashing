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

func TestUpdate(t *testing.T) {
	m := New[int, string]()

	m.Set(1, "apple")
	val, ok := m.Get(1)
	if !ok {
		t.Error("Ok should be true for key '1' stored in the map.")
	}
	if val != "apple" {
		t.Errorf("Val mapped to key '1' was %s. Expected 'apple'", val)
	}

	m.Set(1, "banana")
	if m.Len() != 1 {
		t.Errorf("Map should only contain 1 element. Found %d", m.Len())
	}
	val, ok = m.Get(1)
	if !ok {
		t.Error("Ok should be true for key '1' stored in the map.")
	}
	if val != "banana" {
		t.Errorf("Val mapped to key '1' was %s. Expected 'banana'", val)
	}
}

