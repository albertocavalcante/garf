package pointer

import "testing"

func TestRef(t *testing.T) {
	type T int

	val := T(0)
	pointer := To(val)

	if *pointer != val {
		t.Errorf("expected %d, got %d", val, *pointer)
	}

	val = T(1)
	pointer = To(val)

	if *pointer != val {
		t.Errorf("expected %d, got %d", val, *pointer)
	}
}

func TestDeref(t *testing.T) {
	type T int

	var val, def T = 1, 0

	out := Deref(&val, def)
	if out != val {
		t.Errorf("expected %d, got %d", val, out)
	}

	out = Deref(nil, def)
	if out != def {
		t.Errorf("expected %d, got %d", def, out)
	}
}
