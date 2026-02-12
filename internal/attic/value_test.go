package attic

import "testing"

func TestScalarOps(t *testing.T) {
	check := func(got, want Value) {
		t.Helper()
		if got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	check(True.Not(), False)
	check(False.Not(), True)
	check(Unknown.Not(), Unknown)
	check(Both.Not(), Both)

	check(True.And(True), True)
	check(True.And(False), False)
	check(True.And(Unknown), Unknown)
	check(False.And(Unknown), False)
	check(Unknown.And(Unknown), Unknown)

	check(False.Or(False), False)
	check(False.Or(True), True)
	check(False.Or(Unknown), Unknown)
	check(True.Or(Unknown), True)
	check(Unknown.Or(Unknown), Unknown)

	check(True.Implies(True), True)
	check(True.Implies(False), False)
	check(True.Implies(Unknown), Unknown)
	check(False.Implies(False), True)
	check(False.Implies(True), True)
	check(False.Implies(Unknown), True)
	check(Unknown.Implies(True), True)
	check(Unknown.Implies(False), Unknown)
	check(Unknown.Implies(Unknown), Unknown)
}

func TestToBool(t *testing.T) {
	val, ok := True.ToBool()
	if !ok || !val {
		t.Errorf("True.ToBool() = (%v, %v), want (true, true)", val, ok)
	}
	val, ok = False.ToBool()
	if !ok || val {
		t.Errorf("False.ToBool() = (%v, %v), want (false, true)", val, ok)
	}
	val, ok = Unknown.ToBool()
	if ok {
		t.Errorf("Unknown.ToBool() = (%v, %v), want (_, false)", val, ok)
	}
	val, ok = Both.ToBool()
	if ok {
		t.Errorf("Both.ToBool() = (%v, %v), want (_, false)", val, ok)
	}
}

func TestString(t *testing.T) {
	tests := []struct {
		v    Value
		want string
	}{
		{True, "True"},
		{False, "False"},
		{Unknown, "Unknown"},
		{Both, "Both"},
	}
	for _, tt := range tests {
		if got := tt.v.String(); got != tt.want {
			t.Errorf("%d.String() = %q, want %q", tt.v, got, tt.want)
		}
	}
}

func TestScalarAndTruthTable(t *testing.T) {
	expected := [4][4]Value{
		//           Unknown  True     False    Both
		/* Unknown */ {Unknown, Unknown, False, False},
		/* True    */ {Unknown, True, False, Both},
		/* False   */ {False, False, False, False},
		/* Both    */ {False, Both, False, Both},
	}
	variants := [4]Value{Unknown, True, False, Both}
	for i, a := range variants {
		for j, b := range variants {
			got := a.And(b)
			if got != expected[i][j] {
				t.Errorf("%v.And(%v) = %v, want %v", a, b, got, expected[i][j])
			}
		}
	}
}

func TestScalarOrTruthTable(t *testing.T) {
	expected := [4][4]Value{
		//           Unknown  True  False    Both
		/* Unknown */ {Unknown, True, Unknown, True},
		/* True    */ {True, True, True, True},
		/* False   */ {Unknown, True, False, Both},
		/* Both    */ {True, True, Both, Both},
	}
	variants := [4]Value{Unknown, True, False, Both}
	for i, a := range variants {
		for j, b := range variants {
			got := a.Or(b)
			if got != expected[i][j] {
				t.Errorf("%v.Or(%v) = %v, want %v", a, b, got, expected[i][j])
			}
		}
	}
}

func TestScalarMerge(t *testing.T) {
	check := func(a, b, want Value) {
		t.Helper()
		if got := a.Merge(b); got != want {
			t.Errorf("%v.Merge(%v) = %v, want %v", a, b, got, want)
		}
	}
	check(Unknown, Unknown, Unknown)
	check(Unknown, True, True)
	check(Unknown, False, False)
	check(True, False, Both)
	check(Both, True, Both)
	check(Both, False, Both)
	check(Both, Unknown, Both)
	check(True, True, True)
	check(False, False, False)
}

func TestScalarQueries(t *testing.T) {
	if Unknown.HasInfo() {
		t.Error("Unknown.HasInfo() should be false")
	}
	if !True.HasInfo() {
		t.Error("True.HasInfo() should be true")
	}
	if !False.HasInfo() {
		t.Error("False.HasInfo() should be true")
	}
	if !Both.HasInfo() {
		t.Error("Both.HasInfo() should be true")
	}

	if Unknown.IsDetermined() {
		t.Error("Unknown.IsDetermined() should be false")
	}
	if !True.IsDetermined() {
		t.Error("True.IsDetermined() should be true")
	}
	if !False.IsDetermined() {
		t.Error("False.IsDetermined() should be true")
	}
	if Both.IsDetermined() {
		t.Error("Both.IsDetermined() should be false")
	}

	if Unknown.IsContradicted() {
		t.Error("Unknown.IsContradicted() should be false")
	}
	if True.IsContradicted() {
		t.Error("True.IsContradicted() should be false")
	}
	if False.IsContradicted() {
		t.Error("False.IsContradicted() should be false")
	}
	if !Both.IsContradicted() {
		t.Error("Both.IsContradicted() should be true")
	}
}
