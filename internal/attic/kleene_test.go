package attic

import (
	"testing"
)

func TestVecGetSet(t *testing.T) {
	v := NewKleeneVec(100)
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

func TestVecGetSetWordBoundary(t *testing.T) {
	v := NewKleeneVec(128)
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

func TestSetNegativeIndex(t *testing.T) {
	v := NewKleeneVec(10)
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for negative index")
		}
	}()
	v.Set(-1, True)
}

func TestGetNegativeIndex(t *testing.T) {
	v := NewKleeneVec(10)
	if _, err := v.Get(-1); err != ErrOutOfBounds {
		t.Errorf("expected ErrOutOfBounds, got %v", err)
	}
}

func TestVecAnd(t *testing.T) {
	a := KleeneAllTrue(64)
	b := KleeneAllFalse(64)
	c := a.And(b)
	if !c.IsAllFalse() {
		t.Error("expected all false")
	}
}

func TestVecOr(t *testing.T) {
	a := KleeneAllFalse(64)
	b := KleeneAllTrue(64)
	c := a.Or(b)
	if !c.IsAllTrue() {
		t.Error("expected all true")
	}
}

func TestVecNot(t *testing.T) {
	a := KleeneAllTrue(100)
	b := a.Not()
	if !b.IsAllFalse() {
		t.Error("expected all false")
	}
	c := b.Not()
	if !c.IsAllTrue() {
		t.Error("expected all true")
	}
}

func TestVecUnknownAnd(t *testing.T) {
	a := NewKleeneVec(64) // all unknown
	b := KleeneAllTrue(64)
	c := a.And(b)
	if c.CountUnknown() != 64 {
		t.Errorf("expected 64 unknown, got %d", c.CountUnknown())
	}

	d := KleeneAllFalse(64)
	e := a.And(d)
	if !e.IsAllFalse() {
		t.Error("expected all false")
	}
}

func TestVecUnknownOr(t *testing.T) {
	a := NewKleeneVec(64) // all unknown
	b := KleeneAllFalse(64)
	c := a.Or(b)
	if c.CountUnknown() != 64 {
		t.Errorf("expected 64 unknown, got %d", c.CountUnknown())
	}

	d := KleeneAllTrue(64)
	e := a.Or(d)
	if !e.IsAllTrue() {
		t.Error("expected all true")
	}
}

func TestVecImplies(t *testing.T) {
	a := KleeneAllFalse(64)
	b := KleeneAllTrue(64)

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
	u := NewKleeneVec(64)
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

func TestVecBinopMismatchedWidths(t *testing.T) {
	short := KleeneAllTrue(10)
	long := KleeneAllFalse(100)

	c := short.And(long)
	if c.Width() != 100 {
		t.Errorf("And width: got %d, want 100", c.Width())
	}
	if !c.IsAllFalse() {
		t.Error("True AND False / Unknown AND False should all be False")
	}

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

	c1 := long.And(short)
	c2 := short.And(long)
	for i := range c1.Width() {
		g1, _ := c1.Get(i)
		g2, _ := c2.Get(i)
		if g1 != g2 {
			t.Errorf("And not commutative at index %d: %v vs %v", i, g1, g2)
		}
	}

	c = short.Implies(long)
	if c.Width() != 100 {
		t.Errorf("Implies width: got %d, want 100", c.Width())
	}
	for i := range 10 {
		got, _ := c.Get(i)
		if got != False {
			t.Errorf("Implies index %d: got %v, want False", i, got)
		}
	}
	for i := 10; i < 100; i++ {
		got, _ := c.Get(i)
		if got != Unknown {
			t.Errorf("Implies index %d: got %v, want Unknown", i, got)
		}
	}
}

func TestVecBinopMismatchedWidthsCrossWord(t *testing.T) {
	short := KleeneAllTrue(30)
	long := KleeneAllTrue(200)

	c := short.And(long)
	if c.Width() != 200 {
		t.Errorf("width: got %d, want 200", c.Width())
	}
	if c.CountTrue() != 30 {
		t.Errorf("true count: got %d, want 30", c.CountTrue())
	}
	if c.CountUnknown() != 170 {
		t.Errorf("unknown count: got %d, want 170", c.CountUnknown())
	}

	c = short.Or(long)
	if !c.IsAllTrue() {
		t.Error("True OR True / Unknown OR True should all be True")
	}
}

func TestCounts(t *testing.T) {
	v := NewKleeneVec(10)
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

func TestGetOutOfBounds(t *testing.T) {
	v := NewKleeneVec(10)
	if _, err := v.Get(10); err != ErrOutOfBounds {
		t.Errorf("expected ErrOutOfBounds, got %v", err)
	}
	if _, err := v.Get(100); err != ErrOutOfBounds {
		t.Errorf("expected ErrOutOfBounds, got %v", err)
	}
}

func TestSetAutoGrows(t *testing.T) {
	v := NewKleeneVec(10)
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

func TestTruncate(t *testing.T) {
	v := KleeneAllTrue(100)
	v.Truncate(100)
	if v.Width() != 100 || !v.IsAllTrue() {
		t.Error("truncate(100) should be no-op")
	}
	v.Truncate(200)
	if v.Width() != 100 || !v.IsAllTrue() {
		t.Error("truncate(200) should be no-op")
	}

	v = KleeneAllTrue(100)
	v.Truncate(0)
	if v.Width() != 0 || v.CountTrue() != 0 {
		t.Error("truncate(0) failed")
	}

	v = KleeneAllTrue(64)
	v.Truncate(30)
	if v.Width() != 30 || !v.IsAllTrue() || v.CountTrue() != 30 {
		t.Errorf("truncate(30) failed: width=%d allTrue=%v count=%d", v.Width(), v.IsAllTrue(), v.CountTrue())
	}

	v = KleeneAllTrue(200)
	v.Truncate(65)
	if v.Width() != 65 || !v.IsAllTrue() || v.CountTrue() != 65 {
		t.Error("truncate(65) failed")
	}
}

func TestResize(t *testing.T) {
	v := KleeneAllTrue(10)
	v.Resize(100, Unknown)
	if v.Width() != 100 || v.CountTrue() != 10 || v.CountUnknown() != 90 {
		t.Error("resize with Unknown fill failed")
	}

	v = KleeneAllTrue(10)
	v.Resize(100, False)
	if v.Width() != 100 || v.CountTrue() != 10 || v.CountFalse() != 90 || v.CountUnknown() != 0 {
		t.Error("resize with False fill failed")
	}

	v = NewKleeneVec(10)
	v.Resize(100, True)
	if v.Width() != 100 || v.CountUnknown() != 10 || v.CountTrue() != 90 {
		t.Error("resize with True fill failed")
	}

	v = KleeneAllFalse(60)
	v.Resize(200, True)
	if v.Width() != 200 || v.CountFalse() != 60 || v.CountTrue() != 140 {
		t.Errorf("resize across boundary failed: width=%d false=%d true=%d", v.Width(), v.CountFalse(), v.CountTrue())
	}

	v = KleeneAllTrue(100)
	v.Resize(10, False)
	if v.Width() != 10 || !v.IsAllTrue() {
		t.Error("resize shrink failed")
	}

	v = NewKleeneVec(0)
	v.Resize(64, True)
	if v.Width() != 64 || !v.IsAllTrue() {
		t.Error("resize from empty (True) failed")
	}

	v = NewKleeneVec(0)
	v.Resize(100, False)
	if v.Width() != 100 || !v.IsAllFalse() {
		t.Error("resize from empty (False) failed")
	}
}
