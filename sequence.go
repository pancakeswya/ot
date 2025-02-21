package ot

import (
	"errors"
	"unicode/utf8"
	"strings"
)

type Sequence struct {
	ops       []Operation
	baseLen   int
	targetLen int
}

var ErrIncompatibleLengths = errors.New("incompatible lengths")

func NewSequence() *Sequence {
	return &Sequence{}
}

func (seq *Sequence) Delete(n uint64) {
	if n == 0 {
		return
	}
	seq.baseLen += int(n)

	if len(seq.ops) > 0 {
		lastIdx := len(seq.ops) - 1
		if last, ok := seq.ops[lastIdx].(Delete); ok {
			seq.ops[lastIdx] = Delete{N: last.N + n}
			return
		}
	}
	seq.ops = append(seq.ops, Delete{N: n})
}

func (seq *Sequence) Insert(s string) {
	n := utf8.RuneCountInString(s)
	if n == 0 {
		return
	}
	seq.targetLen += n

	if len(seq.ops) > 0 {
		lastIdx := len(seq.ops) - 1
		switch lastOp := seq.ops[lastIdx].(type) {
		case Insert:
			seq.ops[lastIdx] = Insert{Str: lastOp.Str + s}
			return
		case Delete:
			if len(seq.ops) >= 2 {
				preLastIdx := lastIdx - 1
				if preLast, ok := seq.ops[preLastIdx].(Insert); ok {
					seq.ops[preLastIdx] = Insert{Str: preLast.Str + s}
					return
				}
			}
			seq.ops[lastIdx] = Insert{Str: s}
			seq.ops = append(seq.ops, lastOp)
			return
		}
	}
	seq.ops = append(seq.ops, Insert{Str: s})
}

func (seq *Sequence) Retain(n uint64) {
	if n == 0 {
		return
	}
	seq.baseLen += int(n)
	seq.targetLen += int(n)

	if len(seq.ops) > 0 {
		lastIdx := len(seq.ops) - 1
		if last, ok := seq.ops[lastIdx].(Retain); ok {
			seq.ops[lastIdx] = Retain{N: last.N + n}
			return
		}
	}
	seq.ops = append(seq.ops, Retain{N: n})
}

func (seq *Sequence) Apply(s string) (string, error) {
	if utf8.RuneCountInString(s) != seq.baseLen {
		return "", ErrIncompatibleLengths
	}

	var result strings.Builder
	chars := []rune(s)
	pos := 0

	for _, op := range seq.ops {
		switch opVal := op.(type) {
		case Retain:
			for i := uint64(0); i < opVal.N; i++ {
				result.WriteRune(chars[pos])
				pos++
			}
		case Delete:
			pos += int(opVal.N)
		case Insert:
			result.WriteString(opVal.Str)
		}
	}

	return result.String(), nil
}

func (seq *Sequence) IsNoop() bool {
	if len(seq.ops) == 0 {
		return true
	}
	if len(seq.ops) == 1 {
		_, ok := seq.ops[0].(Retain)
		return ok
	}
	return false
}

func (seq *Sequence) Compose(other *Sequence) (*Sequence, error) {
	if seq.targetLen != other.baseLen {
		return nil, ErrIncompatibleLengths
	}

	newOp := NewSequence()
	ops1 := cloneOps(seq.ops)
	ops2 := cloneOps(other.ops)
	i1, i2 := 0, 0

	for i1 < len(ops1) || i2 < len(ops2) {
		if i1 < len(ops1) {
			if op, ok := ops1[i1].(Delete); ok {
				newOp.Delete(op.N)
				i1++
				continue
			}
		}
		if i2 < len(ops2) {
			if op, ok := ops2[i2].(Insert); ok {
				newOp.Insert(op.Str)
				i2++
				continue
			}
		}
		if i1 >= len(ops1) || i2 >= len(ops2) {
			return nil, ErrIncompatibleLengths
		}
		switch op1 := ops1[i1].(type) {
		case Retain:
			switch op2 := ops2[i2].(type) {
			case Retain:
				if op1.N < op2.N {
					newOp.Retain(op1.N)
					ops2[i2] = Retain{N: op2.N - op1.N}
					i1++
				} else if op1.N == op2.N {
					newOp.Retain(op1.N)
					i1++
					i2++
				} else {
					newOp.Retain(op2.N)
					ops1[i1] = Retain{N: op1.N - op2.N}
					i2++
				}
			case Delete:
				if op1.N < op2.N {
					newOp.Delete(op1.N)
					ops2[i2] = Delete{N: op2.N - op1.N}
					i1++
				} else if op1.N == op2.N {
					newOp.Delete(op2.N)
					i1++
					i2++
				} else {
					newOp.Delete(op2.N)
					ops1[i1] = Retain{N: op1.N - op2.N}
					i2++
				}
			}
		case Insert:
			s := []rune(op1.Str)
			n := uint64(len(s))

			switch op2 := ops2[i2].(type) {
			case Delete:
				if n < op2.N {
					ops2[i2] = Delete{N: op2.N - n}
					i1++
				} else if n == op2.N {
					i1++
					i2++
				} else {
					ops1[i1] = Insert{Str: string(s[op2.N:])}
					i2++
				}
			case Retain:
				if n < op2.N {
					newOp.Insert(op1.Str)
					ops2[i2] = Retain{N: op2.N - n}
					i1++
				} else if n == op2.N {
					newOp.Insert(op1.Str)
					i1++
					i2++
				} else {
					newOp.Insert(string(s[:op2.N]))
					ops1[i1] = Insert{Str: string(s[op2.N:])}
					i2++
				}
			}
		}
	}

	return newOp, nil
}

func (seq *Sequence) Transform(other *Sequence) (*Sequence, *Sequence, error) {
	if seq.baseLen != other.baseLen {
		return nil, nil, ErrIncompatibleLengths
	}

	aPrime := NewSequence()
	bPrime := NewSequence()

	ops1 := cloneOps(seq.ops)
	ops2 := cloneOps(other.ops)
	i1, i2 := 0, 0

	for i1 < len(ops1) || i2 < len(ops2) {
		if i1 < len(ops1) {
			if op, ok := ops1[i1].(Insert); ok {
				aPrime.Insert(op.Str)
				bPrime.Retain(uint64(utf8.RuneCountInString(op.Str)))
				i1++
				continue
			}
		}
		if i2 < len(ops2) {
			if op, ok := ops2[i2].(Insert); ok {
				aPrime.Retain(uint64(utf8.RuneCountInString(op.Str)))
				bPrime.Insert(op.Str)
				i2++
				continue
			}
		}
		if i1 >= len(ops1) || i2 >= len(ops2) {
			return nil, nil, ErrIncompatibleLengths
		}
		switch op1 := ops1[i1].(type) {
		case Retain:
			switch op2 := ops2[i2].(type) {
			case Retain:
				if op1.N < op2.N {
					aPrime.Retain(op1.N)
					bPrime.Retain(op1.N)
					ops2[i2] = Retain{N: op2.N - op1.N}
					i1++
				} else if op1.N == op2.N {
					aPrime.Retain(op1.N)
					bPrime.Retain(op1.N)
					i1++
					i2++
				} else {
					aPrime.Retain(op2.N)
					bPrime.Retain(op2.N)
					ops1[i1] = Retain{N: op1.N - op2.N}
					i2++
				}
			case Delete:
				if op1.N < op2.N {
					bPrime.Delete(op1.N)
					ops2[i2] = Delete{N: op2.N - op1.N}
					i1++
				} else if op1.N == op2.N {
					bPrime.Delete(op1.N)
					i1++
					i2++
				} else {
					bPrime.Delete(op2.N)
					ops1[i1] = Retain{N: op1.N - op2.N}
					i2++
				}
			}
		case Delete:
			switch op2 := ops2[i2].(type) {
			case Retain:
				if op1.N < op2.N {
					aPrime.Delete(op1.N)
					ops2[i2] = Retain{N: op2.N - op1.N}
					i1++
				} else if op1.N == op2.N {
					aPrime.Delete(op1.N)
					i1++
					i2++
				} else {
					aPrime.Delete(op2.N)
					ops1[i1] = Delete{N: op1.N - op2.N}
					i2++
				}
			case Delete:
				if op1.N < op2.N {
					ops2[i2] = Delete{N: op2.N - op1.N}
					i1++
				} else if op1.N == op2.N {
					i1++
					i2++
				} else {
					ops1[i1] = Delete{N: op1.N - op2.N}
					i2++
				}
			}
		}
	}
	return aPrime, bPrime, nil
}

func (seq *Sequence) Invert(s string) *Sequence {
	inverse := NewSequence()
	chars := []rune(s)
	pos := 0

	for _, op := range seq.ops {
		switch opVal := op.(type) {
		case Retain:
			inverse.Retain(opVal.N)
			pos += int(opVal.N)
		case Insert:
			inverse.Delete(uint64(utf8.RuneCountInString(opVal.Str)))
		case Delete:
			inverse.Insert(string(chars[pos : pos+int(opVal.N)]))
			pos += int(opVal.N)
		}
	}

	return inverse
}
