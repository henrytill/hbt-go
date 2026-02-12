// Package kleene implements Kleene's three-valued logic: a scalar type and a packed bitvector.
package kleene

import (
	"errors"
	"math/bits"
)

// Kleene represents a single Kleene truth value.
//
// Encoding: (known_bit << 1) | value_bit
//
//	Unknown = 0b00 (known=0, value=0)
//	False   = 0b10 (known=1, value=0)
//	True    = 0b11 (known=1, value=1)
type Kleene uint8

const (
	Unknown Kleene = 0b00
	False   Kleene = 0b10
	True    Kleene = 0b11
)

var fromBits = [4]Kleene{
	Unknown, // 0b00
	Unknown, // 0b01: impossible by invariant
	False,   // 0b10
	True,    // 0b11
}

// IsKnown reports whether the value is True or False (not Unknown).
func (k Kleene) IsKnown() bool {
	return k&0b10 != 0
}

// ToBool returns the boolean value and whether it is known.
// If !ok, the val return is meaningless.
func (k Kleene) ToBool() (val bool, ok bool) {
	if k.IsKnown() {
		return k&1 != 0, true
	}
	return false, false
}

// Not returns the Kleene negation.
func (k Kleene) Not() Kleene {
	switch k {
	case True:
		return False
	case False:
		return True
	default:
		return Unknown
	}
}

// And returns the Kleene conjunction.
func (k Kleene) And(other Kleene) Kleene {
	switch {
	case k == True:
		return other
	case k == False || (k == Unknown && other == False):
		return False
	default:
		return Unknown
	}
}

// Or returns the Kleene disjunction.
func (k Kleene) Or(other Kleene) Kleene {
	switch {
	case k == False:
		return other
	case k == True || (k == Unknown && other == True):
		return True
	default:
		return Unknown
	}
}

// Implies returns the Kleene material implication (¬k ∨ other).
func (k Kleene) Implies(other Kleene) Kleene {
	return k.Not().Or(other)
}

func (k Kleene) String() string {
	switch k {
	case True:
		return "True"
	case False:
		return "False"
	default:
		return "Unknown"
	}
}

// ErrOutOfBounds is returned when an index exceeds the vector width.
var ErrOutOfBounds = errors.New("index out of bounds")

const (
	bitsLog2 = 6
	bitsMask = (1 << bitsLog2) - 1
)

func wordsNeeded(n int) int {
	return (n + bitsMask) >> bitsLog2
}

func tailMask(n int) uint64 {
	r := n & bitsMask
	if r == 0 {
		return ^uint64(0)
	}
	return (1 << r) - 1
}

// Vec is a packed Kleene bitvector using an interleaved two-bitplane layout:
// [known_0, value_0, known_1, value_1, ...].
//
// Invariant: value bits are a subset of known bits within every pair.
// Unused high bits in the last word pair are always zero.
type Vec struct {
	width int
	words []uint64
}

// NewVec creates a vector of width elements, all Unknown.
func NewVec(width int) *Vec {
	nw := wordsNeeded(width)
	return &Vec{
		width: width,
		words: make([]uint64, 2*nw),
	}
}

// AllTrue creates a vector of width elements, all True.
func AllTrue(width int) *Vec {
	nw := wordsNeeded(width)
	words := make([]uint64, 2*nw)
	for i := range words {
		words[i] = ^uint64(0)
	}
	v := &Vec{width: width, words: words}
	v.maskTail()
	return v
}

// AllFalse creates a vector of width elements, all False.
func AllFalse(width int) *Vec {
	nw := wordsNeeded(width)
	words := make([]uint64, 2*nw)
	for i := 0; i < nw; i++ {
		words[2*i] = ^uint64(0) // known
		words[2*i+1] = 0        // value
	}
	v := &Vec{width: width, words: words}
	v.maskTail()
	return v
}

// Width returns the number of elements.
func (v *Vec) Width() int {
	return v.width
}

func (v *Vec) maskTail() {
	nw := wordsNeeded(v.width)
	if nw > 0 {
		m := tailMask(v.width)
		base := 2 * (nw - 1)
		v.words[base] &= m
		v.words[base+1] &= m
	}
}

// Truncate reduces the width. No-op if newWidth >= current width.
func (v *Vec) Truncate(newWidth int) {
	if newWidth >= v.width {
		return
	}
	v.width = newWidth
	nw := wordsNeeded(newWidth)
	v.words = v.words[:2*nw]
	v.maskTail()
}

// Resize changes the width, filling new positions with fill.
// Shrinking delegates to Truncate.
func (v *Vec) Resize(newWidth int, fill Kleene) {
	if newWidth <= v.width {
		v.Truncate(newWidth)
		return
	}
	oldWidth := v.width
	oldNw := wordsNeeded(oldWidth)
	newNw := wordsNeeded(newWidth)

	var fillKnown, fillValue uint64
	switch fill {
	case False:
		fillKnown = ^uint64(0)
	case True:
		fillKnown = ^uint64(0)
		fillValue = ^uint64(0)
	}

	// Fill remaining bits in the current last word pair.
	if oldNw > 0 && oldWidth&bitsMask != 0 {
		highMask := ^tailMask(oldWidth)
		base := 2 * (oldNw - 1)
		v.words[base] |= fillKnown & highMask
		v.words[base+1] |= fillValue & highMask
	}

	// Grow by appending interleaved pairs.
	for i := oldNw; i < newNw; i++ {
		v.words = append(v.words, fillKnown, fillValue)
	}
	v.width = newWidth
	if fill.IsKnown() {
		v.maskTail()
	}
}

// Get returns the value at index i, or ErrOutOfBounds.
func (v *Vec) Get(i int) (Kleene, error) {
	if i < 0 || i >= v.width {
		return Unknown, ErrOutOfBounds
	}
	return v.getUnchecked(i), nil
}

func (v *Vec) getUnchecked(i int) Kleene {
	w := i >> bitsLog2
	b := uint(i & bitsMask)
	base := 2 * w
	knownBit := (v.words[base] >> b) & 1
	valueBit := (v.words[base+1] >> b) & 1
	return fromBits[knownBit<<1|valueBit]
}

// Set sets the value at index i, auto-growing the vector if necessary.
func (v *Vec) Set(i int, val Kleene) {
	if i >= v.width {
		newWidth := i + 1
		newNw := wordsNeeded(newWidth)
		oldNw := wordsNeeded(v.width)
		for j := oldNw; j < newNw; j++ {
			v.words = append(v.words, 0, 0)
		}
		v.width = newWidth
	}
	v.setUnchecked(i, val)
}

func (v *Vec) setUnchecked(i int, val Kleene) {
	w := i >> bitsLog2
	b := uint(i & bitsMask)
	base := 2 * w
	switch val {
	case True:
		v.words[base] |= 1 << b
		v.words[base+1] |= 1 << b
	case False:
		v.words[base] |= 1 << b
		v.words[base+1] &^= 1 << b
	default: // Unknown
		v.words[base] &^= 1 << b
		v.words[base+1] &^= 1 << b
	}
}

func (v *Vec) checkWidth(other *Vec) {
	if v.width != other.width {
		panic("kleene: Vec width mismatch")
	}
}

// Not returns a new vector with each element negated.
func (v *Vec) Not() *Vec {
	out := make([]uint64, len(v.words))
	for i := 0; i < len(v.words); i += 2 {
		k := v.words[i]
		val := v.words[i+1]
		out[i] = k
		out[i+1] = k &^ val
	}
	return &Vec{width: v.width, words: out}
}

// And returns the element-wise Kleene conjunction.
func (v *Vec) And(other *Vec) *Vec {
	v.checkWidth(other)
	out := make([]uint64, len(v.words))
	for i := 0; i < len(v.words); i += 2 {
		ak, av := v.words[i], v.words[i+1]
		bk, bv := other.words[i], other.words[i+1]
		resultTrue := (ak & av) & (bk & bv)
		resultFalse := (ak &^ av) | (bk &^ bv)
		out[i] = resultTrue | resultFalse
		out[i+1] = resultTrue
	}
	return &Vec{width: v.width, words: out}
}

// Or returns the element-wise Kleene disjunction.
func (v *Vec) Or(other *Vec) *Vec {
	v.checkWidth(other)
	out := make([]uint64, len(v.words))
	for i := 0; i < len(v.words); i += 2 {
		ak, av := v.words[i], v.words[i+1]
		bk, bv := other.words[i], other.words[i+1]
		resultTrue := (ak & av) | (bk & bv)
		resultFalse := (ak &^ av) & (bk &^ bv)
		out[i] = resultTrue | resultFalse
		out[i+1] = resultTrue
	}
	return &Vec{width: v.width, words: out}
}

// Implies returns the element-wise Kleene material implication.
func (v *Vec) Implies(other *Vec) *Vec {
	return v.Not().Or(other)
}

// IsAllKnown reports whether every element is True or False.
func (v *Vec) IsAllKnown() bool {
	nw := wordsNeeded(v.width)
	if nw == 0 {
		return true
	}
	m := tailMask(v.width)
	for i := 0; i < nw-1; i++ {
		if v.words[2*i] != ^uint64(0) {
			return false
		}
	}
	return v.words[2*(nw-1)] == m
}

// IsAllTrue reports whether every element is True.
func (v *Vec) IsAllTrue() bool {
	nw := wordsNeeded(v.width)
	if nw == 0 {
		return true
	}
	m := tailMask(v.width)
	for i := 0; i < nw-1; i++ {
		base := 2 * i
		if v.words[base] != ^uint64(0) || v.words[base+1] != ^uint64(0) {
			return false
		}
	}
	base := 2 * (nw - 1)
	return v.words[base] == m && v.words[base+1] == m
}

// IsAllFalse reports whether every element is False.
func (v *Vec) IsAllFalse() bool {
	if !v.IsAllKnown() {
		return false
	}
	for i := 0; i < len(v.words); i += 2 {
		if v.words[i+1] != 0 {
			return false
		}
	}
	return true
}

// CountTrue returns the number of True elements.
func (v *Vec) CountTrue() int {
	n := 0
	for i := 1; i < len(v.words); i += 2 {
		n += bits.OnesCount64(v.words[i])
	}
	return n
}

// CountFalse returns the number of False elements.
func (v *Vec) CountFalse() int {
	n := 0
	for i := 0; i < len(v.words); i += 2 {
		n += bits.OnesCount64(v.words[i] &^ v.words[i+1])
	}
	return n
}

// CountUnknown returns the number of Unknown elements.
func (v *Vec) CountUnknown() int {
	return v.width - v.CountTrue() - v.CountFalse()
}
