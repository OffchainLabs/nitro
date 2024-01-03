package threadsafe

import (
	"errors"
	"testing"
)

func TestNewMap(t *testing.T) {
	m := NewMap[int, string]()
	if !m.IsEmpty() {
		t.Errorf("Expected map to be empty")
	}
}

func TestNewMapFromItems(t *testing.T) {
	initialItems := map[int]string{1: "one", 2: "two"}
	m := NewMapFromItems(initialItems)

	if m.NumItems() != 2 {
		t.Errorf("Expected 2 items, got %d", m.NumItems())
	}

	if val, ok := m.TryGet(1); !ok || val != "one" {
		t.Errorf("Expected 'one', got %s", val)
	}

	if val, ok := m.TryGet(2); !ok || val != "two" {
		t.Errorf("Expected 'two', got %s", val)
	}
}

func TestPutAndGet(t *testing.T) {
	m := NewMap[int, string]()
	m.Put(1, "one")
	if val := m.Get(1); val != "one" {
		t.Errorf("Expected 'one', got %s", val)
	}
}

func TestHas(t *testing.T) {
	m := NewMap[int, string]()
	m.Put(1, "one")
	if !m.Has(1) {
		t.Errorf("Expected key to exist")
	}
}

func TestNumItems(t *testing.T) {
	m := NewMap[int, string]()
	m.Put(1, "one")
	m.Put(2, "two")
	if m.NumItems() != 2 {
		t.Errorf("Expected 2 items, got %d", m.NumItems())
	}
}

func TestTryGet(t *testing.T) {
	m := NewMap[int, string]()
	m.Put(1, "one")
	val, ok := m.TryGet(1)
	if !ok || val != "one" {
		t.Errorf("Expected 'one', got %s", val)
	}
}

func TestDelete(t *testing.T) {
	m := NewMap[int, string]()
	m.Put(1, "one")
	m.Delete(1)
	if m.Has(1) {
		t.Errorf("Expected key to be deleted")
	}
}

func TestForEach(t *testing.T) {
	m := NewMap[int, string]()
	m.Put(1, "one")
	m.Put(2, "two")
	err := m.ForEach(func(k int, v string) error {
		if v == "three" {
			return errors.New("should not have 'three'")
		}
		return nil
	})
	if err != nil {
		t.Errorf("ForEach errored: %v", err)
	}
}
