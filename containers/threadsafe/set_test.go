package threadsafe

import (
	"testing"
)

func TestNewSet(t *testing.T) {
	s := NewSet[int]()
	if s.NumItems() != 0 {
		t.Errorf("Expected 0 items, got %d", s.NumItems())
	}
}

func TestInsert(t *testing.T) {
	s := NewSet[int]()
	s.Insert(1)
	if s.NumItems() != 1 {
		t.Errorf("Expected 1 item, got %d", s.NumItems())
	}
}

func TestHasSet(t *testing.T) {
	s := NewSet[int]()
	s.Insert(1)
	if !s.Has(1) {
		t.Errorf("Expected item to exist")
	}
}

func TestDeleteSet(t *testing.T) {
	s := NewSet[int]()
	s.Insert(1)
	s.Delete(1)
	if s.Has(1) {
		t.Errorf("Expected item to be deleted")
	}
}

func TestNumItemsSet(t *testing.T) {
	s := NewSet[int]()
	s.Insert(1)
	s.Insert(2)
	if s.NumItems() != 2 {
		t.Errorf("Expected 2 items, got %d", s.NumItems())
	}
}

func TestForEachSet(t *testing.T) {
	s := NewSet[int]()
	s.Insert(1)
	s.Insert(2)
	count := 0
	s.ForEach(func(elem int) {
		count++
	})
	if count != 2 {
		t.Errorf("Expected to iterate 2 times, iterated %d times", count)
	}
}
