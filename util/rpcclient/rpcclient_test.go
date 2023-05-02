package rpcclient

import "testing"

func TestLogArgs(t *testing.T) {
	t.Parallel()

	str := logArgs(0, 1, 2, 3, "hello, world")
	if str != "[1, 2, 3, hello, world]" {
		t.Fatal("unexpected logs limit 0 got:", str)
	}

	str = logArgs(100, 1, 2, 3, "hello, world")
	if str != "[1, 2, 3, hello, world]" {
		t.Fatal("unexpected logs limit 100 got:", str)
	}

	str = logArgs(4, 1, 2, 3, "hello, world")
	if str != "[1, 2, 3, h...d]" {
		t.Fatal("unexpected logs limit 4 got:", str)
	}

}
