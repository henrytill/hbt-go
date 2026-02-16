package belnap

import "errors"

// Value represents a single truth value in Belnap's four-valued logic.
//
// Encoding: (neg_bit << 1) | pos_bit
//
//	Unknown = 0b00 (pos=0, neg=0)
//	True    = 0b01 (pos=1, neg=0)
//	False   = 0b10 (pos=0, neg=1)
//	Both    = 0b11 (pos=1, neg=1)
type Value uint8

const (
	Unknown Value = 0b00
	True    Value = 0b01
	False   Value = 0b10
	Both    Value = 0b11
)

var fromBits = [4]Value{Unknown, True, False, Both}

var ErrOutOfBounds = errors.New("index out of bounds")

func (v Value) IsKnown() bool {
	return v != Unknown
}

func (v Value) IsDetermined() bool {
	return (v&1)^(v>>1) != 0
}

func (v Value) IsContradicted() bool {
	return v == Both
}

func (v Value) ToBool() (val bool, ok bool) {
	if v.IsDetermined() {
		return v&1 != 0, true
	}
	return false, false
}

func (v Value) Not() Value {
	return fromBits[(uint8(v)>>1)|((uint8(v)&1)<<1)]
}

func (v Value) And(other Value) Value {
	rPos := (uint8(v) & 1) & (uint8(other) & 1)
	rNeg := (uint8(v) >> 1) | (uint8(other) >> 1)
	return fromBits[rNeg<<1|rPos]
}

func (v Value) Or(other Value) Value {
	rPos := (uint8(v) & 1) | (uint8(other) & 1)
	rNeg := (uint8(v) >> 1) & (uint8(other) >> 1)
	return fromBits[rNeg<<1|rPos]
}

func (v Value) Implies(other Value) Value {
	return v.Not().Or(other)
}

func (v Value) Merge(other Value) Value {
	return fromBits[uint8(v)|uint8(other)]
}

func (v Value) String() string {
	switch v {
	case True:
		return "True"
	case False:
		return "False"
	case Both:
		return "Both"
	default:
		return "Unknown"
	}
}
