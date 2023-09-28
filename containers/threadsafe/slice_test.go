package threadsafe

import (
	"testing"
)

func TestNewSlice(t *testing.T) {
	s := NewSlice[int]()
	if s.Len() != 0 {
		t.Errorf("Expected length to be 0, got %d", s.Len())
	}
}

func TestPush(t *testing.T) {
	s := NewSlice[int]()
	s.Push(1)
	if s.Len() != 1 {
		t.Errorf("Expected length to be 1, got %d", s.Len())
	}
}

func TestLen(t *testing.T) {
	s := NewSlice[int]()
	s.Push(1)
	s.Push(2)
	if s.Len() != 2 {
		t.Errorf("Expected length to be 2, got %d", s.Len())
	}
}

func TestFind(t *testing.T) {
	s := NewSlice[int]()
	s.Push(1)
	s.Push(2)
	s.Push(3)

	found := s.Find(func(idx int, elem int) bool {
		return elem == 2
	})

	if !found {
		t.Errorf("Expected to find the element")
	}

	notFound := s.Find(func(idx int, elem int) bool {
		return elem == 4
	})

	if notFound {
		t.Errorf("Did not expect to find the element")
	}
}
