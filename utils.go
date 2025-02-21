package ot

import (
	"math/rand"
	"unicode/utf8"
)

func genString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func genSequence(s string) *Sequence {
	op := NewSequence()
	for {
		left := utf8.RuneCountInString(s) - op.baseLen
		if left == 0 {
			break
		}
		i := 1
		if left != 1 {
			i += randInt(0, minInt(left-1, 20))
		}
		if f := randFloat(0.0, 1.0); f < 0.2 {
			op.Insert(genString(i))
		} else if f < 0.4 {
			op.Delete(uint64(i))
		} else {
			op.Retain(uint64(i))
		}
	}
	if randFloat(0.0, 1.0) < 0.3 {
		op.Insert("1" + genString(10))
	}
	return op
}

func randInt(min int, max int) int {
	return min + rand.Intn(max-min)
}

func randFloat(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func cloneOps(ops []Operation) []Operation {
	return append(ops[:0:0], ops...)
}
