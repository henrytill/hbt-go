package belnap

import "testing"

func TestVecGetSetAllFour(t *testing.T) {
	v := NewVec(4)
	v.Set(0, Unknown)
	v.Set(1, True)
	v.Set(2, False)
	v.Set(3, Both)
	if got, _ := v.Get(0); got != Unknown {
		t.Errorf("index 0: got %v, want Unknown", got)
	}
	if got, _ := v.Get(1); got != True {
		t.Errorf("index 1: got %v, want True", got)
	}
	if got, _ := v.Get(2); got != False {
		t.Errorf("index 2: got %v, want False", got)
	}
	if got, _ := v.Get(3); got != Both {
		t.Errorf("index 3: got %v, want Both", got)
	}
}

func TestVecBulkAnd(t *testing.T) {
	a := AllTrue(64)
	b := AllFalse(64)
	c := a.And(b)
	if !c.IsAllFalse() {
		t.Error("expected all false")
	}
}

func TestVecBulkOr(t *testing.T) {
	a := AllFalse(64)
	b := AllTrue(64)
	c := a.Or(b)
	if !c.IsAllTrue() {
		t.Error("expected all true")
	}
}

func TestVecBulkNot(t *testing.T) {
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

func TestVecBulkMerge(t *testing.T) {
	a := AllTrue(64)
	b := AllFalse(64)
	c := a.Merge(b)
	if c.CountBoth() != 64 {
		t.Errorf("expected 64 both, got %d", c.CountBoth())
	}
	if c.CountTrue() != 0 {
		t.Errorf("expected 0 true, got %d", c.CountTrue())
	}
	if c.CountFalse() != 0 {
		t.Errorf("expected 0 false, got %d", c.CountFalse())
	}
	if c.CountUnknown() != 0 {
		t.Errorf("expected 0 unknown, got %d", c.CountUnknown())
	}
}

func TestVecBulkConsensus(t *testing.T) {
	a := AllTrue(64)
	b := AllFalse(64)
	c := a.Consensus(b)
	if c.CountUnknown() != 64 {
		t.Errorf("expected 64 unknown, got %d", c.CountUnknown())
	}
	if c.CountTrue() != 0 || c.CountFalse() != 0 || c.CountBoth() != 0 {
		t.Errorf("expected all unknown, got T=%d F=%d B=%d",
			c.CountTrue(), c.CountFalse(), c.CountBoth())
	}
}

func TestVecConsensusDifferentWidths(t *testing.T) {
	short := NewVec(10)
	short.Set(0, True)
	short.Set(1, Both)
	short.Set(2, Both)

	long := NewVec(100)
	long.Set(0, True)
	long.Set(1, True)
	long.Set(2, False)
	long.Set(99, Both)

	ab := short.Consensus(long)
	ba := long.Consensus(short)
	if ab.Width() != 100 || ba.Width() != 100 {
		t.Errorf("width: got %d/%d, want 100", ab.Width(), ba.Width())
	}

	// True consensus True = True
	if got, _ := ab.Get(0); got != True {
		t.Errorf("index 0: got %v, want True", got)
	}
	// Both consensus True = True
	if got, _ := ab.Get(1); got != True {
		t.Errorf("index 1: got %v, want True", got)
	}
	// Both consensus False = False
	if got, _ := ab.Get(2); got != False {
		t.Errorf("index 2: got %v, want False", got)
	}
	// Unknown (short) consensus Both (long) = Unknown
	if got, _ := ab.Get(99); got != Unknown {
		t.Errorf("index 99: got %v, want Unknown", got)
	}
	// Beyond short: Unknown consensus Unknown = Unknown
	if got, _ := ab.Get(50); got != Unknown {
		t.Errorf("index 50: got %v, want Unknown", got)
	}

	for i := range ab.Width() {
		g1, _ := ab.Get(i)
		g2, _ := ba.Get(i)
		if g1 != g2 {
			t.Errorf("Consensus not commutative at index %d: %v vs %v", i, g1, g2)
		}
	}
}

func TestVecIsConsistent(t *testing.T) {
	a := AllTrue(64)
	if !a.IsConsistent() {
		t.Error("all-true should be consistent")
	}

	b := NewVec(10)
	b.Set(0, True)
	b.Set(1, False)
	if !b.IsConsistent() {
		t.Error("true+false should be consistent")
	}

	b.Set(2, Both)
	if b.IsConsistent() {
		t.Error("vec with Both should not be consistent")
	}
}

func TestVecIsAllDetermined(t *testing.T) {
	v := NewVec(4)
	v.Set(0, True)
	v.Set(1, False)
	v.Set(2, True)
	v.Set(3, False)
	if !v.IsAllDetermined() {
		t.Error("all True/False should be determined")
	}

	v.Set(3, Unknown)
	if v.IsAllDetermined() {
		t.Error("vec with Unknown should not be all determined")
	}

	v.Set(3, Both)
	if v.IsAllDetermined() {
		t.Error("vec with Both should not be all determined")
	}
}

func TestVecCounts(t *testing.T) {
	v := NewVec(10)
	v.Set(0, True)
	v.Set(1, True)
	v.Set(2, False)
	v.Set(3, Both)
	if v.CountTrue() != 2 {
		t.Errorf("expected 2 true, got %d", v.CountTrue())
	}
	if v.CountFalse() != 1 {
		t.Errorf("expected 1 false, got %d", v.CountFalse())
	}
	if v.CountBoth() != 1 {
		t.Errorf("expected 1 both, got %d", v.CountBoth())
	}
	if v.CountUnknown() != 6 {
		t.Errorf("expected 6 unknown, got %d", v.CountUnknown())
	}
}

func TestVecWordBoundaries(t *testing.T) {
	// Element 63: bit 63 (sign bit) of word-pair 0.
	v := NewVec(65)
	v.Set(63, Both)
	if got, _ := v.Get(63); got != Both {
		t.Errorf("get 63: got %v, want Both", got)
	}
	if got, _ := v.Get(62); got != Unknown {
		t.Errorf("get 62: got %v, want Unknown", got)
	}
	if got, _ := v.Get(64); got != Unknown {
		t.Errorf("get 64: got %v, want Unknown", got)
	}

	// Element 64: bit 0 of word-pair 1.
	v.Set(64, True)
	if got, _ := v.Get(64); got != True {
		t.Errorf("get 64: got %v, want True", got)
	}
	if got, _ := v.Get(63); got != Both {
		t.Errorf("get 63 after setting 64: got %v, want Both", got)
	}
}

func TestVecWidth63(t *testing.T) {
	// width=63 exercises r=63 in tailMask, the largest non-aligned width.
	v := AllTrue(63)
	if !v.IsAllTrue() {
		t.Error("expected IsAllTrue")
	}
	if !v.IsAllDetermined() {
		t.Error("expected IsAllDetermined")
	}
	if !v.IsConsistent() {
		t.Error("expected IsConsistent")
	}
	if got, _ := v.Get(62); got != True {
		t.Errorf("get 62: got %v, want True", got)
	}
	merged := v.Merge(AllFalse(63))
	if merged.CountBoth() != 63 {
		t.Errorf("count_both after merge: got %d, want 63", merged.CountBoth())
	}
}

func TestVecAutoGrow(t *testing.T) {
	v := NewVec(10)
	v.Set(100, Both)
	if v.Width() != 101 {
		t.Errorf("expected width 101, got %d", v.Width())
	}
	if got, _ := v.Get(100); got != Both {
		t.Errorf("expected Both at 100, got %v", got)
	}
	if got, _ := v.Get(50); got != Unknown {
		t.Errorf("expected Unknown at 50, got %v", got)
	}
	if _, err := v.Get(200); err != ErrOutOfBounds {
		t.Errorf("expected ErrOutOfBounds at 200, got %v", err)
	}
}

func TestVecResize(t *testing.T) {
	// grow with Unknown fill
	v := AllTrue(10)
	v.Resize(100, Unknown)
	if v.Width() != 100 || v.CountTrue() != 10 || v.CountUnknown() != 90 {
		t.Error("resize with Unknown fill failed")
	}

	// grow with Both fill
	v = AllTrue(10)
	v.Resize(100, Both)
	if v.Width() != 100 || v.CountTrue() != 10 || v.CountBoth() != 90 {
		t.Error("resize with Both fill failed")
	}

	// grow with False fill
	v = AllTrue(10)
	v.Resize(100, False)
	if v.Width() != 100 || v.CountTrue() != 10 || v.CountFalse() != 90 {
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

func TestVecTruncate(t *testing.T) {
	v := AllTrue(100)
	v.Truncate(100)
	if v.Width() != 100 || !v.IsAllTrue() {
		t.Error("truncate(100) should be no-op")
	}

	v = AllTrue(200)
	v.Truncate(65)
	if v.Width() != 65 || !v.IsAllTrue() || v.CountTrue() != 65 {
		t.Error("truncate(65) failed")
	}
}

func TestVecAndDifferentWidths(t *testing.T) {
	short := NewVec(10)
	short.Set(0, True)
	short.Set(1, False)
	short.Set(2, Both)

	long := NewVec(100)
	long.Set(0, True)
	long.Set(1, True)
	long.Set(2, True)
	long.Set(99, True)

	ab := short.And(long)
	ba := long.And(short)
	if ab.Width() != 100 || ba.Width() != 100 {
		t.Errorf("width: got %d/%d, want 100", ab.Width(), ba.Width())
	}

	// True & True = True
	if got, _ := ab.Get(0); got != True {
		t.Errorf("index 0: got %v, want True", got)
	}
	// False & True = False
	if got, _ := ab.Get(1); got != False {
		t.Errorf("index 1: got %v, want False", got)
	}
	// Both & True = Both
	if got, _ := ab.Get(2); got != Both {
		t.Errorf("index 2: got %v, want Both", got)
	}
	// Unknown (short) & True (long) = Unknown
	if got, _ := ab.Get(99); got != Unknown {
		t.Errorf("index 99: got %v, want Unknown", got)
	}
	// Beyond short: Unknown & Unknown = Unknown
	if got, _ := ab.Get(50); got != Unknown {
		t.Errorf("index 50: got %v, want Unknown", got)
	}

	// Commutativity check
	for i := range ab.Width() {
		g1, _ := ab.Get(i)
		g2, _ := ba.Get(i)
		if g1 != g2 {
			t.Errorf("And not commutative at index %d: %v vs %v", i, g1, g2)
		}
	}
}

func TestVecOrDifferentWidths(t *testing.T) {
	short := NewVec(10)
	short.Set(0, True)
	short.Set(1, False)
	short.Set(2, Both)

	long := NewVec(100)
	long.Set(0, False)
	long.Set(1, True)
	long.Set(2, False)
	long.Set(99, False)

	ab := short.Or(long)
	ba := long.Or(short)
	if ab.Width() != 100 || ba.Width() != 100 {
		t.Errorf("width: got %d/%d, want 100", ab.Width(), ba.Width())
	}

	// True | False = True
	if got, _ := ab.Get(0); got != True {
		t.Errorf("index 0: got %v, want True", got)
	}
	// False | True = True
	if got, _ := ab.Get(1); got != True {
		t.Errorf("index 1: got %v, want True", got)
	}
	// Both | False = Both
	if got, _ := ab.Get(2); got != Both {
		t.Errorf("index 2: got %v, want Both", got)
	}
	// Unknown (short) | False (long) = Unknown
	if got, _ := ab.Get(99); got != Unknown {
		t.Errorf("index 99: got %v, want Unknown", got)
	}
	// Beyond short: Unknown | Unknown = Unknown
	if got, _ := ab.Get(50); got != Unknown {
		t.Errorf("index 50: got %v, want Unknown", got)
	}

	// Commutativity check
	for i := range ab.Width() {
		g1, _ := ab.Get(i)
		g2, _ := ba.Get(i)
		if g1 != g2 {
			t.Errorf("Or not commutative at index %d: %v vs %v", i, g1, g2)
		}
	}
}

func TestVecMergeDifferentWidths(t *testing.T) {
	short := NewVec(10)
	short.Set(0, True)
	short.Set(1, False)

	long := NewVec(100)
	long.Set(0, False)
	long.Set(1, True)
	long.Set(99, True)

	ab := short.Merge(long)
	ba := long.Merge(short)
	if ab.Width() != 100 || ba.Width() != 100 {
		t.Errorf("width: got %d/%d, want 100", ab.Width(), ba.Width())
	}

	// True merge False = Both
	if got, _ := ab.Get(0); got != Both {
		t.Errorf("index 0: got %v, want Both", got)
	}
	// False merge True = Both
	if got, _ := ab.Get(1); got != Both {
		t.Errorf("index 1: got %v, want Both", got)
	}
	// Unknown (short) merge True (long) = True
	if got, _ := ab.Get(99); got != True {
		t.Errorf("index 99: got %v, want True", got)
	}
	// Beyond short: Unknown merge Unknown = Unknown
	if got, _ := ab.Get(50); got != Unknown {
		t.Errorf("index 50: got %v, want Unknown", got)
	}

	// Commutativity check
	for i := range ab.Width() {
		g1, _ := ab.Get(i)
		g2, _ := ba.Get(i)
		if g1 != g2 {
			t.Errorf("Merge not commutative at index %d: %v vs %v", i, g1, g2)
		}
	}
}

func TestVecImpliesDifferentWidths(t *testing.T) {
	short := AllTrue(10)
	long := AllTrue(100)
	result := short.Implies(long)
	if result.Width() != 100 {
		t.Errorf("width: got %d, want 100", result.Width())
	}
	// True -> True = True for first 10
	if got, _ := result.Get(0); got != True {
		t.Errorf("index 0: got %v, want True", got)
	}
	// Unknown -> True = True for positions beyond short
	if got, _ := result.Get(50); got != True {
		t.Errorf("index 50: got %v, want True", got)
	}
}
