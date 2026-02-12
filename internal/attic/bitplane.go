package attic

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

func andWord(aPos, aNeg, bPos, bNeg uint64) (uint64, uint64) {
	return aPos & bPos, aNeg | bNeg
}

func orWord(aPos, aNeg, bPos, bNeg uint64) (uint64, uint64) {
	return aPos | bPos, aNeg & bNeg
}

func notWord(pos, neg uint64) (uint64, uint64) {
	return neg, pos
}

func mergeWord(aPos, aNeg, bPos, bNeg uint64) (uint64, uint64) {
	return aPos | bPos, aNeg | bNeg
}
