package attic

import "math/bits"

// BelnapVec is a packed bitvector using an interleaved two-bitplane layout:
// [pos_0, neg_0, pos_1, neg_1, ...].
//
// All four Value states (Unknown, True, False, Both) are valid at every position.
type BelnapVec struct {
	width int
	words []uint64
}

func NewBelnapVec(width int) *BelnapVec {
	nw := wordsNeeded(width)
	return &BelnapVec{
		width: width,
		words: make([]uint64, 2*nw),
	}
}

func newBelnapFilled(width int, fill Value) *BelnapVec {
	nw := wordsNeeded(width)
	words := make([]uint64, 2*nw)
	fillPos := ^uint64(0) * uint64(fill&1)
	fillNeg := ^uint64(0) * uint64(fill>>1)
	for i := range nw {
		words[2*i] = fillPos
		words[2*i+1] = fillNeg
	}
	v := &BelnapVec{width: width, words: words}
	v.maskTail()
	return v
}

func BelnapAllTrue(width int) *BelnapVec  { return newBelnapFilled(width, True) }
func BelnapAllFalse(width int) *BelnapVec { return newBelnapFilled(width, False) }

func (v *BelnapVec) Width() int {
	return v.width
}

func (v *BelnapVec) maskTail() {
	nw := wordsNeeded(v.width)
	if nw > 0 {
		m := tailMask(v.width)
		base := 2 * (nw - 1)
		v.words[base] &= m
		v.words[base+1] &= m
	}
}

func (v *BelnapVec) Truncate(newWidth int) {
	if newWidth >= v.width {
		return
	}
	v.width = newWidth
	nw := wordsNeeded(newWidth)
	v.words = v.words[:2*nw]
	v.maskTail()
}

func (v *BelnapVec) Resize(newWidth int, fill Value) {
	if newWidth <= v.width {
		v.Truncate(newWidth)
		return
	}
	oldWidth := v.width
	oldNw := wordsNeeded(oldWidth)
	newNw := wordsNeeded(newWidth)

	fillPos := ^uint64(0) * uint64(fill&1)
	fillNeg := ^uint64(0) * uint64(fill>>1)

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

func (v *BelnapVec) Get(i int) (Value, error) {
	if i < 0 || i >= v.width {
		return Unknown, ErrOutOfBounds
	}
	return v.getUnchecked(i), nil
}

func (v *BelnapVec) getUnchecked(i int) Value {
	w := i >> bitsLog2
	b := uint(i & bitsMask)
	base := 2 * w
	posBit := (v.words[base] >> b) & 1
	negBit := (v.words[base+1] >> b) & 1
	return fromBits[negBit<<1|posBit]
}

func (v *BelnapVec) Set(i int, val Value) {
	if i < 0 {
		panic("attic: negative index")
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

func (v *BelnapVec) setUnchecked(i int, val Value) {
	w := i >> bitsLog2
	b := uint(i & bitsMask)
	base := 2 * w
	mask := uint64(1) << b
	if val&1 != 0 {
		v.words[base] |= mask
	} else {
		v.words[base] &^= mask
	}
	if val>>1 != 0 {
		v.words[base+1] |= mask
	} else {
		v.words[base+1] &^= mask
	}
}

func (v *BelnapVec) Not() *BelnapVec {
	out := make([]uint64, len(v.words))
	for i := 0; i < len(v.words); i += 2 {
		out[i], out[i+1] = notWord(v.words[i], v.words[i+1])
	}
	r := &BelnapVec{width: v.width, words: out}
	r.maskTail()
	return r
}

func (v *BelnapVec) And(other *BelnapVec) *BelnapVec {
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
	return &BelnapVec{width: width, words: out}
}

func (v *BelnapVec) Or(other *BelnapVec) *BelnapVec {
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
	return &BelnapVec{width: width, words: out}
}

func (v *BelnapVec) Implies(other *BelnapVec) *BelnapVec {
	return v.Not().Or(other)
}

func (v *BelnapVec) Merge(other *BelnapVec) *BelnapVec {
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
		out[base], out[base+1] = mergeWord(aPos, aNeg, bPos, bNeg)
	}
	return &BelnapVec{width: width, words: out}
}

func (v *BelnapVec) IsConsistent() bool {
	for i := 0; i < len(v.words); i += 2 {
		if v.words[i]&v.words[i+1] != 0 {
			return false
		}
	}
	return true
}

func (v *BelnapVec) IsAllDetermined() bool {
	nw := wordsNeeded(v.width)
	if nw == 0 {
		return true
	}
	m := tailMask(v.width)
	for i := 0; i < nw-1; i++ {
		base := 2 * i
		if v.words[base]^v.words[base+1] != ^uint64(0) {
			return false
		}
	}
	base := 2 * (nw - 1)
	return v.words[base]^v.words[base+1] == m
}

func (v *BelnapVec) IsAllTrue() bool {
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

func (v *BelnapVec) IsAllFalse() bool {
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

func (v *BelnapVec) CountTrue() int {
	n := 0
	for i := 0; i < len(v.words); i += 2 {
		n += bits.OnesCount64(v.words[i] &^ v.words[i+1])
	}
	return n
}

func (v *BelnapVec) CountFalse() int {
	n := 0
	for i := 0; i < len(v.words); i += 2 {
		n += bits.OnesCount64(v.words[i+1] &^ v.words[i])
	}
	return n
}

func (v *BelnapVec) CountBoth() int {
	n := 0
	for i := 0; i < len(v.words); i += 2 {
		n += bits.OnesCount64(v.words[i] & v.words[i+1])
	}
	return n
}

func (v *BelnapVec) CountUnknown() int {
	return v.width - v.CountTrue() - v.CountFalse() - v.CountBoth()
}

