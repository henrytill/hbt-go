package attic

import "math/bits"

// KleeneVec is a packed bitvector using an interleaved two-bitplane layout:
// [pos_0, neg_0, pos_1, neg_1, ...].
//
// Invariant: pos & neg == 0 within every word pair (no position is Both).
// Unused high bits in the last word pair are always zero.
type KleeneVec struct {
	width int
	words []uint64
}

func NewKleeneVec(width int) *KleeneVec {
	nw := wordsNeeded(width)
	return &KleeneVec{
		width: width,
		words: make([]uint64, 2*nw),
	}
}

func newKleeneFilled(width int, fill Value) *KleeneVec {
	nw := wordsNeeded(width)
	words := make([]uint64, 2*nw)
	var fillPos, fillNeg uint64
	switch fill {
	case True:
		fillPos = ^uint64(0)
	case False:
		fillNeg = ^uint64(0)
	}
	for i := range nw {
		words[2*i] = fillPos
		words[2*i+1] = fillNeg
	}
	v := &KleeneVec{width: width, words: words}
	v.maskTail()
	return v
}

func KleeneAllTrue(width int) *KleeneVec  { return newKleeneFilled(width, True) }
func KleeneAllFalse(width int) *KleeneVec { return newKleeneFilled(width, False) }

func (v *KleeneVec) Width() int {
	return v.width
}

func (v *KleeneVec) wordsRaw() []uint64 {
	return v.words
}

func kleeneVecFromRawParts(width int, words []uint64) *KleeneVec {
	return &KleeneVec{width: width, words: words}
}

func (v *KleeneVec) maskTail() {
	nw := wordsNeeded(v.width)
	if nw > 0 {
		m := tailMask(v.width)
		base := 2 * (nw - 1)
		v.words[base] &= m
		v.words[base+1] &= m
	}
}

func (v *KleeneVec) Truncate(newWidth int) {
	if newWidth >= v.width {
		return
	}
	v.width = newWidth
	nw := wordsNeeded(newWidth)
	v.words = v.words[:2*nw]
	v.maskTail()
}

func (v *KleeneVec) Resize(newWidth int, fill Value) {
	if fill == Both {
		panic("attic: KleeneVec.Resize called with Both")
	}
	if newWidth <= v.width {
		v.Truncate(newWidth)
		return
	}
	oldWidth := v.width
	oldNw := wordsNeeded(oldWidth)
	newNw := wordsNeeded(newWidth)

	var fillPos, fillNeg uint64
	switch fill {
	case True:
		fillPos = ^uint64(0)
	case False:
		fillNeg = ^uint64(0)
	}

	if oldNw > 0 && oldWidth&bitsMask != 0 {
		highMask := ^tailMask(oldWidth)
		base := 2 * (oldNw - 1)
		v.words[base] |= fillPos & highMask
		v.words[base+1] |= fillNeg & highMask
	}

	for i := oldNw; i < newNw; i++ {
		v.words = append(v.words, fillPos, fillNeg)
	}
	v.width = newWidth
	if fill.HasInfo() {
		v.maskTail()
	}
}

func (v *KleeneVec) Get(i int) (Value, error) {
	if i < 0 || i >= v.width {
		return Unknown, ErrOutOfBounds
	}
	return v.getUnchecked(i), nil
}

func (v *KleeneVec) getUnchecked(i int) Value {
	w := i >> bitsLog2
	b := uint(i & bitsMask)
	base := 2 * w
	posBit := (v.words[base] >> b) & 1
	negBit := (v.words[base+1] >> b) & 1
	return fromBits[negBit<<1|posBit]
}

func (v *KleeneVec) Set(i int, val Value) {
	if i < 0 {
		panic("attic: negative index")
	}
	if val == Both {
		panic("attic: KleeneVec.Set called with Both")
	}
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

func (v *KleeneVec) setUnchecked(i int, val Value) {
	w := i >> bitsLog2
	b := uint(i & bitsMask)
	base := 2 * w
	switch val {
	case True:
		v.words[base] |= 1 << b
		v.words[base+1] &^= 1 << b
	case False:
		v.words[base] &^= 1 << b
		v.words[base+1] |= 1 << b
	default: // Unknown
		v.words[base] &^= 1 << b
		v.words[base+1] &^= 1 << b
	}
}

func (v *KleeneVec) Not() *KleeneVec {
	out := make([]uint64, len(v.words))
	for i := 0; i < len(v.words); i += 2 {
		pos := v.words[i]
		neg := v.words[i+1]
		out[i] = neg
		out[i+1] = pos
	}
	r := &KleeneVec{width: v.width, words: out}
	r.maskTail()
	return r
}

func (v *KleeneVec) And(other *KleeneVec) *KleeneVec {
	width := max(v.width, other.width)
	nw := wordsNeeded(width)
	out := make([]uint64, 2*nw)
	for i := range nw {
		base := 2 * i
		var aPos, aNeg, bPos, bNeg uint64
		if base+1 < len(v.words) {
			aPos, aNeg = v.words[base], v.words[base+1]
		}
		if base+1 < len(other.words) {
			bPos, bNeg = other.words[base], other.words[base+1]
		}
		out[base], out[base+1] = andWord(aPos, aNeg, bPos, bNeg)
	}
	return &KleeneVec{width: width, words: out}
}

func (v *KleeneVec) Or(other *KleeneVec) *KleeneVec {
	width := max(v.width, other.width)
	nw := wordsNeeded(width)
	out := make([]uint64, 2*nw)
	for i := range nw {
		base := 2 * i
		var aPos, aNeg, bPos, bNeg uint64
		if base+1 < len(v.words) {
			aPos, aNeg = v.words[base], v.words[base+1]
		}
		if base+1 < len(other.words) {
			bPos, bNeg = other.words[base], other.words[base+1]
		}
		out[base], out[base+1] = orWord(aPos, aNeg, bPos, bNeg)
	}
	return &KleeneVec{width: width, words: out}
}

func (v *KleeneVec) Implies(other *KleeneVec) *KleeneVec {
	return v.Not().Or(other)
}

func (v *KleeneVec) IsAllKnown() bool {
	nw := wordsNeeded(v.width)
	if nw == 0 {
		return true
	}
	m := tailMask(v.width)
	for i := 0; i < nw-1; i++ {
		if v.words[2*i]|v.words[2*i+1] != ^uint64(0) {
			return false
		}
	}
	base := 2 * (nw - 1)
	return v.words[base]|v.words[base+1] == m
}

func (v *KleeneVec) IsAllTrue() bool {
	nw := wordsNeeded(v.width)
	if nw == 0 {
		return true
	}
	m := tailMask(v.width)
	for i := 0; i < nw-1; i++ {
		base := 2 * i
		if v.words[base] != ^uint64(0) || v.words[base+1] != 0 {
			return false
		}
	}
	base := 2 * (nw - 1)
	return v.words[base] == m && v.words[base+1] == 0
}

func (v *KleeneVec) IsAllFalse() bool {
	nw := wordsNeeded(v.width)
	if nw == 0 {
		return true
	}
	m := tailMask(v.width)
	for i := 0; i < nw-1; i++ {
		base := 2 * i
		if v.words[base] != 0 || v.words[base+1] != ^uint64(0) {
			return false
		}
	}
	base := 2 * (nw - 1)
	return v.words[base] == 0 && v.words[base+1] == m
}

func (v *KleeneVec) CountTrue() int {
	n := 0
	for i := 0; i < len(v.words); i += 2 {
		n += bits.OnesCount64(v.words[i])
	}
	return n
}

func (v *KleeneVec) CountFalse() int {
	n := 0
	for i := 0; i < len(v.words); i += 2 {
		n += bits.OnesCount64(v.words[i+1])
	}
	return n
}

func (v *KleeneVec) CountUnknown() int {
	return v.width - v.CountTrue() - v.CountFalse()
}
