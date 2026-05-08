package belnap

import (
	"math/rand"
	"reflect"
	"slices"
	"testing"
	"testing/quick"
)

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

func TestVecToSliceFromSliceRoundtrip(t *testing.T) {
	if got := NewVec(0).ToSlice(); len(got) != 0 {
		t.Errorf("empty: got %v, want []", got)
	}

	xs := []Value{Unknown, True, False, Both}
	if got := FromSlice(xs).ToSlice(); !slices.Equal(got, xs) {
		t.Errorf("4 elems: got %v, want %v", got, xs)
	}

	// 64 elements exercises exactly one full word-pair.
	xs64 := make([]Value, 64)
	for i := range xs64 {
		xs64[i] = True
	}
	if got := AllTrue(64).ToSlice(); !slices.Equal(got, xs64) {
		t.Errorf("64 elems: got %v, want all True", got)
	}

	// 65 elements: last element straddles into word-pair 1.
	xs65 := make([]Value, 65)
	for i := range xs65 {
		xs65[i] = True
	}
	xs65[64] = False
	if got := FromSlice(xs65).ToSlice(); !slices.Equal(got, xs65) {
		t.Errorf("65 elems: got %v, want %v", got, xs65)
	}
}

func TestVecAll(t *testing.T) {
	xs := []Value{Unknown, True, False, Both}
	v := FromSlice(xs)
	var indices []int
	var values []Value
	for i, val := range v.All() {
		indices = append(indices, i)
		values = append(values, val)
	}
	if !slices.Equal(indices, []int{0, 1, 2, 3}) {
		t.Errorf("indices: got %v, want [0 1 2 3]", indices)
	}
	if !slices.Equal(values, xs) {
		t.Errorf("values: got %v, want %v", values, xs)
	}

	// Early termination via break.
	count := 0
	for range v.All() {
		count++
		if count == 2 {
			break
		}
	}
	if count != 2 {
		t.Errorf("early break: got %d iterations, want 2", count)
	}
}

func TestVecFindFirst(t *testing.T) {
	v := FromSlice([]Value{False, False, True, Both})
	if i, ok := v.FindFirst(True); !ok || i != 2 {
		t.Errorf("first True: got (%d, %v), want (2, true)", i, ok)
	}
	if i, ok := v.FindFirst(False); !ok || i != 0 {
		t.Errorf("first False: got (%d, %v), want (0, true)", i, ok)
	}
	if i, ok := v.FindFirst(Both); !ok || i != 3 {
		t.Errorf("first Both: got (%d, %v), want (3, true)", i, ok)
	}
	if _, ok := v.FindFirst(Unknown); ok {
		t.Errorf("first Unknown: got ok=true, want false")
	}

	// Empty vec.
	if _, ok := NewVec(0).FindFirst(True); ok {
		t.Error("empty: got ok=true, want false")
	}

	// Match at word boundary (index 64, word-pair 1).
	xs := make([]Value, 65)
	for i := range xs {
		xs[i] = False
	}
	xs[64] = True
	if i, ok := FromSlice(xs).FindFirst(True); !ok || i != 64 {
		t.Errorf("word-boundary True: got (%d, %v), want (64, true)", i, ok)
	}

	// FindFirst Unknown across multiple words: ensure tail-mask doesn't
	// produce a false hit on garbage bits past width.
	v63 := AllTrue(63)
	if _, ok := v63.FindFirst(Unknown); ok {
		t.Error("AllTrue(63) FindFirst Unknown: got ok=true, want false")
	}
}

func TestVecEqual(t *testing.T) {
	a := FromSlice([]Value{True, False, Both})
	b := FromSlice([]Value{True, False, Both})
	if !a.Equal(b) {
		t.Error("identical vecs: Equal returned false")
	}

	c := FromSlice([]Value{True, False, Unknown})
	if a.Equal(c) {
		t.Error("differing values: Equal returned true")
	}

	d := FromSlice([]Value{True, False})
	if a.Equal(d) {
		t.Error("differing widths: Equal returned true")
	}

	if !NewVec(0).Equal(NewVec(0)) {
		t.Error("empty vecs: Equal returned false")
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

const propMaxN = 200

func randXs(r *rand.Rand, n int) []Value {
	xs := make([]Value, n)
	for i := range xs {
		xs[i] = Value(r.Intn(4))
	}
	return xs
}

type vec1 struct{ Xs []Value }

func (vec1) Generate(r *rand.Rand, _ int) reflect.Value {
	n := r.Intn(propMaxN + 1)
	return reflect.ValueOf(vec1{Xs: randXs(r, n)})
}

type vec2 struct{ Xs, Ys []Value }

func (vec2) Generate(r *rand.Rand, _ int) reflect.Value {
	n := r.Intn(propMaxN + 1)
	return reflect.ValueOf(vec2{Xs: randXs(r, n), Ys: randXs(r, n)})
}

type vec3 struct{ Xs, Ys, Zs []Value }

func (vec3) Generate(r *rand.Rand, _ int) reflect.Value {
	n := r.Intn(propMaxN + 1)
	return reflect.ValueOf(vec3{Xs: randXs(r, n), Ys: randXs(r, n), Zs: randXs(r, n)})
}

type vecGetSet struct {
	Xs []Value
	I  int
	V  Value
}

func (vecGetSet) Generate(r *rand.Rand, _ int) reflect.Value {
	n := r.Intn(propMaxN) + 1
	return reflect.ValueOf(vecGetSet{
		Xs: randXs(r, n),
		I:  r.Intn(n),
		V:  Value(r.Intn(4)),
	})
}

type vecNeedle struct {
	Needle Value
	Xs     []Value
}

func (vecNeedle) Generate(r *rand.Rand, _ int) reflect.Value {
	n := r.Intn(propMaxN + 1)
	return reflect.ValueOf(vecNeedle{
		Needle: Value(r.Intn(4)),
		Xs:     randXs(r, n),
	})
}

func runProp(t *testing.T, f any) {
	t.Helper()
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestPropOrCommutativity(t *testing.T) {
	runProp(t, func(p vec2) bool {
		a, b := FromSlice(p.Xs), FromSlice(p.Ys)
		return a.Or(b).Equal(b.Or(a))
	})
}

func TestPropOrAssociativity(t *testing.T) {
	runProp(t, func(p vec3) bool {
		a, b, c := FromSlice(p.Xs), FromSlice(p.Ys), FromSlice(p.Zs)
		return a.Or(b).Or(c).Equal(a.Or(b.Or(c)))
	})
}

func TestPropOrIdempotency(t *testing.T) {
	runProp(t, func(p vec1) bool {
		a := FromSlice(p.Xs)
		return a.Or(a).Equal(a)
	})
}

func TestPropAndCommutativity(t *testing.T) {
	runProp(t, func(p vec2) bool {
		a, b := FromSlice(p.Xs), FromSlice(p.Ys)
		return a.And(b).Equal(b.And(a))
	})
}

func TestPropAndAssociativity(t *testing.T) {
	runProp(t, func(p vec3) bool {
		a, b, c := FromSlice(p.Xs), FromSlice(p.Ys), FromSlice(p.Zs)
		return a.And(b).And(c).Equal(a.And(b.And(c)))
	})
}

func TestPropAndIdempotency(t *testing.T) {
	runProp(t, func(p vec1) bool {
		a := FromSlice(p.Xs)
		return a.And(a).Equal(a)
	})
}

func TestPropAbsorptionOrAnd(t *testing.T) {
	runProp(t, func(p vec2) bool {
		a, b := FromSlice(p.Xs), FromSlice(p.Ys)
		return a.Or(a.And(b)).Equal(a)
	})
}

func TestPropAbsorptionAndOr(t *testing.T) {
	runProp(t, func(p vec2) bool {
		a, b := FromSlice(p.Xs), FromSlice(p.Ys)
		return a.And(a.Or(b)).Equal(a)
	})
}

func TestPropAndDistributesOverOr(t *testing.T) {
	runProp(t, func(p vec3) bool {
		a, b, c := FromSlice(p.Xs), FromSlice(p.Ys), FromSlice(p.Zs)
		return a.And(b.Or(c)).Equal(a.And(b).Or(a.And(c)))
	})
}

func TestPropOrDistributesOverAnd(t *testing.T) {
	runProp(t, func(p vec3) bool {
		a, b, c := FromSlice(p.Xs), FromSlice(p.Ys), FromSlice(p.Zs)
		return a.Or(b.And(c)).Equal(a.Or(b).And(a.Or(c)))
	})
}

func TestPropOrFalseIdentity(t *testing.T) {
	runProp(t, func(p vec1) bool {
		a := FromSlice(p.Xs)
		return a.Or(AllFalse(len(p.Xs))).Equal(a)
	})
}

func TestPropAndTrueIdentity(t *testing.T) {
	runProp(t, func(p vec1) bool {
		a := FromSlice(p.Xs)
		return a.And(AllTrue(len(p.Xs))).Equal(a)
	})
}

func TestPropOrTrueAnnihilator(t *testing.T) {
	runProp(t, func(p vec1) bool {
		a := FromSlice(p.Xs)
		return a.Or(AllTrue(len(p.Xs))).Equal(AllTrue(len(p.Xs)))
	})
}

func TestPropAndFalseAnnihilator(t *testing.T) {
	runProp(t, func(p vec1) bool {
		a := FromSlice(p.Xs)
		return a.And(AllFalse(len(p.Xs))).Equal(AllFalse(len(p.Xs)))
	})
}

func TestPropImpliesDefinition(t *testing.T) {
	runProp(t, func(p vec2) bool {
		a, b := FromSlice(p.Xs), FromSlice(p.Ys)
		return a.Implies(b).Equal(a.Not().Or(b))
	})
}

func TestPropNotInvolutive(t *testing.T) {
	runProp(t, func(p vec1) bool {
		a := FromSlice(p.Xs)
		return a.Not().Not().Equal(a)
	})
}

func TestPropDeMorganAnd(t *testing.T) {
	runProp(t, func(p vec2) bool {
		a, b := FromSlice(p.Xs), FromSlice(p.Ys)
		return a.And(b).Not().Equal(a.Not().Or(b.Not()))
	})
}

func TestPropDeMorganOr(t *testing.T) {
	runProp(t, func(p vec2) bool {
		a, b := FromSlice(p.Xs), FromSlice(p.Ys)
		return a.Or(b).Not().Equal(a.Not().And(b.Not()))
	})
}

func TestPropMergeCommutativity(t *testing.T) {
	runProp(t, func(p vec2) bool {
		a, b := FromSlice(p.Xs), FromSlice(p.Ys)
		return a.Merge(b).Equal(b.Merge(a))
	})
}

func TestPropMergeAssociativity(t *testing.T) {
	runProp(t, func(p vec3) bool {
		a, b, c := FromSlice(p.Xs), FromSlice(p.Ys), FromSlice(p.Zs)
		return a.Merge(b).Merge(c).Equal(a.Merge(b.Merge(c)))
	})
}

func TestPropMergeIdempotency(t *testing.T) {
	runProp(t, func(p vec1) bool {
		a := FromSlice(p.Xs)
		return a.Merge(a).Equal(a)
	})
}

func TestPropMergeUnknownIdentity(t *testing.T) {
	runProp(t, func(p vec1) bool {
		a := FromSlice(p.Xs)
		return a.Merge(NewVec(len(p.Xs))).Equal(a)
	})
}

func TestPropConsensusCommutativity(t *testing.T) {
	runProp(t, func(p vec2) bool {
		a, b := FromSlice(p.Xs), FromSlice(p.Ys)
		return a.Consensus(b).Equal(b.Consensus(a))
	})
}

func TestPropConsensusAssociativity(t *testing.T) {
	runProp(t, func(p vec3) bool {
		a, b, c := FromSlice(p.Xs), FromSlice(p.Ys), FromSlice(p.Zs)
		return a.Consensus(b).Consensus(c).Equal(a.Consensus(b.Consensus(c)))
	})
}

func TestPropConsensusIdempotency(t *testing.T) {
	runProp(t, func(p vec1) bool {
		a := FromSlice(p.Xs)
		return a.Consensus(a).Equal(a)
	})
}

func TestPropConsensusBothIdentity(t *testing.T) {
	runProp(t, func(p vec1) bool {
		a := FromSlice(p.Xs)
		return a.Consensus(AllBoth(len(p.Xs))).Equal(a)
	})
}

func TestPropCountsSumToWidth(t *testing.T) {
	runProp(t, func(p vec1) bool {
		v := FromSlice(p.Xs)
		return v.CountTrue()+v.CountFalse()+v.CountBoth()+v.CountUnknown() == len(p.Xs)
	})
}

func TestPropIsConsistentIffNoBoth(t *testing.T) {
	runProp(t, func(p vec1) bool {
		v := FromSlice(p.Xs)
		return v.IsConsistent() == (v.CountBoth() == 0)
	})
}

func TestPropIsAllDeterminedIff(t *testing.T) {
	runProp(t, func(p vec1) bool {
		v := FromSlice(p.Xs)
		return v.IsAllDetermined() == (v.CountUnknown() == 0 && v.CountBoth() == 0)
	})
}

func TestPropIsAllTrueIff(t *testing.T) {
	runProp(t, func(p vec1) bool {
		v := FromSlice(p.Xs)
		return v.IsAllTrue() == (v.CountTrue() == len(p.Xs))
	})
}

func TestPropIsAllFalseIff(t *testing.T) {
	runProp(t, func(p vec1) bool {
		v := FromSlice(p.Xs)
		return v.IsAllFalse() == (v.CountFalse() == len(p.Xs))
	})
}

func TestPropFromSliceToSliceRoundtrip(t *testing.T) {
	runProp(t, func(p vec1) bool {
		return slices.Equal(FromSlice(p.Xs).ToSlice(), p.Xs)
	})
}

func TestPropToSliceMatchesGet(t *testing.T) {
	runProp(t, func(p vec1) bool {
		v := FromSlice(p.Xs)
		got := v.ToSlice()
		for i := range p.Xs {
			val, _ := v.Get(i)
			if got[i] != val {
				return false
			}
		}
		return true
	})
}

func TestPropFindFirstReturnsMatch(t *testing.T) {
	runProp(t, func(p vecNeedle) bool {
		v := FromSlice(p.Xs)
		i, ok := v.FindFirst(p.Needle)
		if !ok {
			return true
		}
		got, _ := v.Get(i)
		return got == p.Needle
	})
}

func TestPropFindFirstIsLeftmost(t *testing.T) {
	runProp(t, func(p vecNeedle) bool {
		v := FromSlice(p.Xs)
		i, ok := v.FindFirst(p.Needle)
		if !ok {
			return true
		}
		for j := range i {
			got, _ := v.Get(j)
			if got == p.Needle {
				return false
			}
		}
		return true
	})
}

func TestPropFindFirstNoneIffCountZero(t *testing.T) {
	runProp(t, func(p vec1) bool {
		v := FromSlice(p.Xs)
		for _, c := range []struct {
			needle Value
			count  int
		}{
			{True, v.CountTrue()},
			{False, v.CountFalse()},
			{Both, v.CountBoth()},
			{Unknown, v.CountUnknown()},
		} {
			_, ok := v.FindFirst(c.needle)
			if ok != (c.count > 0) {
				return false
			}
		}
		return true
	})
}

func TestPropGetAfterSet(t *testing.T) {
	runProp(t, func(p vecGetSet) bool {
		v := FromSlice(p.Xs)
		v.Set(p.I, p.V)
		got, _ := v.Get(p.I)
		return got == p.V
	})
}
