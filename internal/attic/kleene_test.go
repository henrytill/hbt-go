package kleene

import (
	"testing"
)

// TestScalarOps verifies the full truth tables for Not, And, Or, and Implies.
func TestScalarOps(t *testing.T) {
	check := func(got, want Kleene) {
		t.Helper()
		if got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	check(True.Not(), False)
	check(False.Not(), True)
	check(Unknown.Not(), Unknown)

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

// TestToBool verifies the Kleene to (bool, bool) conversion for all three values.
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
}

// TestString verifies the string representations of all three Kleene values.
func TestString(t *testing.T) {
	tests := []struct {
		k    Kleene
		want string
	}{
		{True, "True"},
		{False, "False"},
		{Unknown, "Unknown"},
	}
	for _, tt := range tests {
		if got := tt.k.String(); got != tt.want {
			t.Errorf("%d.String() = %q, want %q", tt.k, got, tt.want)
		}
	}
}

// TestVecGetSet verifies basic Get/Set round-trips on a 100-element vector.
func TestVecGetSet(t *testing.T) {
	v := NewVec(100)
	if got, _ := v.Get(0); got != Unknown {
		t.Errorf("expected Unknown, got %v", got)
	}
	v.Set(0, True)
	v.Set(1, False)
	v.Set(99, True)
	if got, _ := v.Get(0); got != True {
		t.Errorf("expected True, got %v", got)
	}
	if got, _ := v.Get(1); got != False {
		t.Errorf("expected False, got %v", got)
	}
	if got, _ := v.Get(2); got != Unknown {
		t.Errorf("expected Unknown, got %v", got)
	}
	if got, _ := v.Get(99); got != True {
		t.Errorf("expected True, got %v", got)
	}
}

// TestVecGetSetWordBoundary verifies Get/Set at the 64-bit word boundary (indices 63, 64, 65).
func TestVecGetSetWordBoundary(t *testing.T) {
	v := NewVec(128)
	v.Set(63, True)
	v.Set(64, False)
	v.Set(65, True)
	if got, _ := v.Get(63); got != True {
		t.Errorf("index 63: expected True, got %v", got)
	}
	if got, _ := v.Get(64); got != False {
		t.Errorf("index 64: expected False, got %v", got)
	}
	if got, _ := v.Get(65); got != True {
		t.Errorf("index 65: expected True, got %v", got)
	}
}

// TestSetNegativeIndex verifies that Set panics on a negative index.
func TestSetNegativeIndex(t *testing.T) {
	v := NewVec(10)
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for negative index")
		}
	}()
	v.Set(-1, True)
}

// TestGetNegativeIndex verifies that Get returns ErrOutOfBounds for a negative index.
func TestGetNegativeIndex(t *testing.T) {
	v := NewVec(10)
	if _, err := v.Get(-1); err != ErrOutOfBounds {
		t.Errorf("expected ErrOutOfBounds, got %v", err)
	}
}

// TestVecAnd verifies element-wise And with True and False vectors.
func TestVecAnd(t *testing.T) {
	a := AllTrue(64)
	b := AllFalse(64)
	c := a.And(b)
	if !c.IsAllFalse() {
		t.Error("expected all false")
	}
}

// TestVecOr verifies element-wise Or with False and True vectors.
func TestVecOr(t *testing.T) {
	a := AllFalse(64)
	b := AllTrue(64)
	c := a.Or(b)
	if !c.IsAllTrue() {
		t.Error("expected all true")
	}
}

// TestVecNot verifies element-wise Not with double negation.
func TestVecNot(t *testing.T) {
	a := AllTrue(100)
	b := a.Not()
	if !b.IsAllFalse() {
		t.Error("expected all false")
	}
	c := b.Not()
	if !c.IsAllTrue() {
		t.Error("expected all true")
	}
}

// TestVecUnknownAnd verifies Unknown propagation in element-wise And.
func TestVecUnknownAnd(t *testing.T) {
	a := NewVec(64) // all unknown
	b := AllTrue(64)
	c := a.And(b)
	if c.CountUnknown() != 64 {
		t.Errorf("expected 64 unknown, got %d", c.CountUnknown())
	}

	d := AllFalse(64)
	e := a.And(d)
	if !e.IsAllFalse() {
		t.Error("expected all false")
	}
}

// TestVecUnknownOr verifies Unknown propagation in element-wise Or.
func TestVecUnknownOr(t *testing.T) {
	a := NewVec(64) // all unknown
	b := AllFalse(64)
	c := a.Or(b)
	if c.CountUnknown() != 64 {
		t.Errorf("expected 64 unknown, got %d", c.CountUnknown())
	}

	d := AllTrue(64)
	e := a.Or(d)
	if !e.IsAllTrue() {
		t.Error("expected all true")
	}
}

// TestVecImplies verifies element-wise Implies with all value combinations.
func TestVecImplies(t *testing.T) {
	a := AllFalse(64)
	b := AllTrue(64)

	// False -> anything = True
	c := a.Implies(b)
	if !c.IsAllTrue() {
		t.Error("False implies True should be all True")
	}
	c = a.Implies(a)
	if !c.IsAllTrue() {
		t.Error("False implies False should be all True")
	}

	// True -> False = False
	c = b.Implies(a)
	if !c.IsAllFalse() {
		t.Error("True implies False should be all False")
	}

	// Unknown -> True = True
	u := NewVec(64)
	c = u.Implies(b)
	if !c.IsAllTrue() {
		t.Error("Unknown implies True should be all True")
	}

	// Unknown -> False = Unknown
	c = u.Implies(a)
	if c.CountUnknown() != 64 {
		t.Errorf("Unknown implies False: expected 64 unknown, got %d", c.CountUnknown())
	}
}

// TestVecBinopMismatchedWidths verifies that And, Or, and Implies work with different-width vectors.
func TestVecBinopMismatchedWidths(t *testing.T) {
	// short And long: short positions use both operands, extended positions treat short as Unknown.
	short := AllTrue(10)
	long := AllFalse(100)

	// True AND False = False for [0,10); Unknown AND False = False for [10,100)
	c := short.And(long)
	if c.Width() != 100 {
		t.Errorf("And width: got %d, want 100", c.Width())
	}
	if !c.IsAllFalse() {
		t.Error("True AND False / Unknown AND False should all be False")
	}

	// True OR False = True for [0,10); Unknown OR False = Unknown for [10,100)
	c = short.Or(long)
	if c.Width() != 100 {
		t.Errorf("Or width: got %d, want 100", c.Width())
	}
	if c.CountTrue() != 10 {
		t.Errorf("Or true count: got %d, want 10", c.CountTrue())
	}
	if c.CountUnknown() != 90 {
		t.Errorf("Or unknown count: got %d, want 90", c.CountUnknown())
	}

	// Commutative check: long op short == short op long
	c1 := long.And(short)
	c2 := short.And(long)
	for i := range c1.Width() {
		g1, _ := c1.Get(i)
		g2, _ := c2.Get(i)
		if g1 != g2 {
			t.Errorf("And not commutative at index %d: %v vs %v", i, g1, g2)
		}
	}

	// Implies across widths
	c = short.Implies(long)
	if c.Width() != 100 {
		t.Errorf("Implies width: got %d, want 100", c.Width())
	}
	// True -> False = False for [0,10)
	for i := range 10 {
		got, _ := c.Get(i)
		if got != False {
			t.Errorf("Implies index %d: got %v, want False", i, got)
		}
	}
	// Unknown -> False = Unknown for [10,100)
	for i := 10; i < 100; i++ {
		got, _ := c.Get(i)
		if got != Unknown {
			t.Errorf("Implies index %d: got %v, want Unknown", i, got)
		}
	}
}

// TestVecBinopMismatchedWidthsCrossWord verifies mismatched-width ops across word boundaries.
func TestVecBinopMismatchedWidthsCrossWord(t *testing.T) {
	short := AllTrue(30)
	long := AllTrue(200)

	c := short.And(long)
	if c.Width() != 200 {
		t.Errorf("width: got %d, want 200", c.Width())
	}
	if c.CountTrue() != 30 {
		t.Errorf("true count: got %d, want 30", c.CountTrue())
	}
	// Unknown AND True = Unknown for [30,200)
	if c.CountUnknown() != 170 {
		t.Errorf("unknown count: got %d, want 170", c.CountUnknown())
	}

	c = short.Or(long)
	if !c.IsAllTrue() {
		t.Error("True OR True / Unknown OR True should all be True")
	}
}

// TestCounts verifies CountTrue, CountFalse, and CountUnknown on a mixed vector.
func TestCounts(t *testing.T) {
	v := NewVec(10)
	v.Set(0, True)
	v.Set(1, True)
	v.Set(2, False)
	if v.CountTrue() != 2 {
		t.Errorf("expected 2 true, got %d", v.CountTrue())
	}
	if v.CountFalse() != 1 {
		t.Errorf("expected 1 false, got %d", v.CountFalse())
	}
	if v.CountUnknown() != 7 {
		t.Errorf("expected 7 unknown, got %d", v.CountUnknown())
	}
}

// TestGetOutOfBounds verifies that Get returns ErrOutOfBounds for indices past the width.
func TestGetOutOfBounds(t *testing.T) {
	v := NewVec(10)
	if _, err := v.Get(10); err != ErrOutOfBounds {
		t.Errorf("expected ErrOutOfBounds, got %v", err)
	}
	if _, err := v.Get(100); err != ErrOutOfBounds {
		t.Errorf("expected ErrOutOfBounds, got %v", err)
	}
}

// TestSetAutoGrows verifies that Set auto-grows the vector when the index exceeds the width.
func TestSetAutoGrows(t *testing.T) {
	v := NewVec(10)
	v.Set(100, True)
	if v.Width() != 101 {
		t.Errorf("expected width 101, got %d", v.Width())
	}
	if got, _ := v.Get(100); got != True {
		t.Errorf("expected True at 100, got %v", got)
	}
	if got, _ := v.Get(50); got != Unknown {
		t.Errorf("expected Unknown at 50, got %v", got)
	}
	if _, err := v.Get(200); err != ErrOutOfBounds {
		t.Errorf("expected ErrOutOfBounds at 200, got %v", err)
	}
}

// TestTruncate verifies width reduction including partial words, no-op cases, and zero width.
func TestTruncate(t *testing.T) {
	// no-op when new_width >= width
	v := AllTrue(100)
	v.Truncate(100)
	if v.Width() != 100 || !v.IsAllTrue() {
		t.Error("truncate(100) should be no-op")
	}
	v.Truncate(200)
	if v.Width() != 100 || !v.IsAllTrue() {
		t.Error("truncate(200) should be no-op")
	}

	// truncate to zero
	v = AllTrue(100)
	v.Truncate(0)
	if v.Width() != 0 || v.CountTrue() != 0 {
		t.Error("truncate(0) failed")
	}

	// partial word
	v = AllTrue(64)
	v.Truncate(30)
	if v.Width() != 30 || !v.IsAllTrue() || v.CountTrue() != 30 {
		t.Errorf("truncate(30) failed: width=%d allTrue=%v count=%d", v.Width(), v.IsAllTrue(), v.CountTrue())
	}

	// across word boundary
	v = AllTrue(200)
	v.Truncate(65)
	if v.Width() != 65 || !v.IsAllTrue() || v.CountTrue() != 65 {
		t.Error("truncate(65) failed")
	}
}

// TestResize verifies grow and shrink with all fill values, including cross-word-boundary growth and growth from empty.
func TestResize(t *testing.T) {
	// grow with Unknown fill
	v := AllTrue(10)
	v.Resize(100, Unknown)
	if v.Width() != 100 || v.CountTrue() != 10 || v.CountUnknown() != 90 {
		t.Error("resize with Unknown fill failed")
	}

	// grow with False fill
	v = AllTrue(10)
	v.Resize(100, False)
	if v.Width() != 100 || v.CountTrue() != 10 || v.CountFalse() != 90 || v.CountUnknown() != 0 {
		t.Error("resize with False fill failed")
	}

	// grow with True fill
	v = NewVec(10)
	v.Resize(100, True)
	if v.Width() != 100 || v.CountUnknown() != 10 || v.CountTrue() != 90 {
		t.Error("resize with True fill failed")
	}

	// grow across word boundary
	v = AllFalse(60)
	v.Resize(200, True)
	if v.Width() != 200 || v.CountFalse() != 60 || v.CountTrue() != 140 {
		t.Errorf("resize across boundary failed: width=%d false=%d true=%d", v.Width(), v.CountFalse(), v.CountTrue())
	}

	// shrink
	v = AllTrue(100)
	v.Resize(10, False)
	if v.Width() != 10 || !v.IsAllTrue() {
		t.Error("resize shrink failed")
	}

	// grow from empty
	v = NewVec(0)
	v.Resize(64, True)
	if v.Width() != 64 || !v.IsAllTrue() {
		t.Error("resize from empty (True) failed")
	}

	v = NewVec(0)
	v.Resize(100, False)
	if v.Width() != 100 || !v.IsAllFalse() {
		t.Error("resize from empty (False) failed")
	}
}
