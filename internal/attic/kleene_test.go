package kleene

import (
	"testing"
)

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
}

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

func TestVecAnd(t *testing.T) {
	a := AllTrue(64)
	b := AllFalse(64)
	c := a.And(b)
	if !c.IsAllFalse() {
		t.Error("expected all false")
	}
}

func TestVecOr(t *testing.T) {
	a := AllFalse(64)
	b := AllTrue(64)
	c := a.Or(b)
	if !c.IsAllTrue() {
		t.Error("expected all true")
	}
}

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

func TestGetOutOfBounds(t *testing.T) {
	v := NewVec(10)
	if _, err := v.Get(10); err != ErrOutOfBounds {
		t.Errorf("expected ErrOutOfBounds, got %v", err)
	}
	if _, err := v.Get(100); err != ErrOutOfBounds {
		t.Errorf("expected ErrOutOfBounds, got %v", err)
	}
}

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
