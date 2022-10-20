package id

import "math/big"

const PendingID = "pending-id"
const MaxStr = "10000000000000000000000000000000000000000000000000000000000000000"
const HalfMaxStr = "8000000000000000000000000000000000000000000000000000000000000000"

var Max, _ = new(big.Int).SetString(MaxStr, 16)
var HalfMax, _ = new(big.Int).SetString(HalfMaxStr, 16)
var Zero = new(big.Int).SetInt64(0)
var IDCache = make(map[string]*big.Int)

func ShortID(i string) string {
	return i[0:6]
}

func IDsToShortIDs(ids []string) []string {
	var shortIDs []string
	for _, i := range ids {
		shortIDs = append(shortIDs, ShortID(i))
	}
	return shortIDs
}

func GetBigInt(s string) *big.Int {
	if s, ok := IDCache[s]; ok {
		return s
	}
	b, _ := new(big.Int).SetString(s, 16)
	IDCache[s] = b
	return b
}

func ParseBigIntBase10(num int64) *big.Int {
	return new(big.Int).SetInt64(num)
}

func BigIntLessThanEqMax(b *big.Int) bool {
	return b.Cmp(Max) <= 0
}

func ClosestIDInList(i string, ids []string) string {
	closest := ids[0]
	distance := Max
	for _, candidateID := range ids {
		candidateDistance := DistanceBetweenIDs(i, candidateID)
		if candidateDistance.Cmp(distance) < 0 {
			distance = candidateDistance
			closest = candidateID
		}
	}
	return closest
}

func DistanceBetweenIDs(a string, b string) *big.Int {
	intA := GetBigInt(a)
	intB := GetBigInt(b)
	var distance *big.Int
	distance.Sub(intB, intA)
	distance.Abs(distance)
	if HalfMax.Cmp(distance) < 0 {
		distance.Sub(Max, distance)
	}
	return distance
}

func DirectedDistanceBetweenIDs(a string, b string) *big.Int {
	intA := GetBigInt(a)
	intB := GetBigInt(b)
	var distance *big.Int
	distance.Sub(intB, intA)
	// distance < 0
	if Zero.Cmp(distance) > 0 {
		distance.Add(Max, distance)
	}
	return distance
}

func BigIntToID(b *big.Int) string {
	res := b.String()
	for {
		if len(res) < len(HalfMaxStr)+1 {
			res = "0" + res
		} else {
			break
		}
	}
	IDCache[res] = b
	return res
}

func IdealFinger(i string, level int) string {
	ideal := GetBigInt(i)
	offset := HalfMax
	for i := 0; i < level; i++ {
		ideal.Add(ideal, offset)
		if ideal.Cmp(Max) > 0 {
			ideal.Sub(ideal, Max)
		}
		// divide offset by 2
		offset.Rsh(offset, 1)
	}
	return BigIntToID(ideal)
}

func CandidateMatchesApproximately(i string, candidate string, level int) bool {
	// how far are we from the ideal candidate
	distance := DistanceBetweenIDs(IdealFinger(i, level), candidate)
	var maxDistance *big.Int
	// our acceptable distance depends on our level within the chord
	maxDistance.Rsh(HalfMax, uint(level+2))
	// check whether our distance < max
	return distance.Cmp(maxDistance) < 0
}

func ShouldYieldToID(a string, b string) bool {
	return DirectedDistanceBetweenIDs(a, b).Cmp(HalfMax) < 0
}
