package pointer_test

import (
	"testing"

	"github.com/albertocavalcante/garf/pkg/pointer"
)

func TestRef(t *testing.T) {
	type T int

	val := T(0)
	ptr := pointer.To(val)

	if *ptr != val {
		t.Errorf("expected %d, got %d", val, *ptr)
	}

	val = T(1)
	ptr = pointer.To(val)

	if *ptr != val {
		t.Errorf("expected %d, got %d", val, *ptr)
	}
}

func TestDeref(t *testing.T) {
	type T int

	var val, def T = 1, 0

	out := pointer.Deref(&val, def)
	if out != val {
		t.Errorf("expected %d, got %d", val, out)
	}

	out = pointer.Deref(nil, def)
	if out != def {
		t.Errorf("expected %d, got %d", def, out)
	}
}
