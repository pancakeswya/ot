package ot

import (
	"errors"
	"unicode/utf8"
	"strings"
	"encoding/json"
	"fmt"
)

type Sequence struct {
	Ops       []Operation
	BaseLen   int
	TargetLen int
}

var ErrIncompatibleLengths = errors.New("incompatible lengths")

func NewSequence() *Sequence {
	return &Sequence{}
}

func (seq *Sequence) Delete(n uint64) {
	if n == 0 {
		return
	}
	seq.BaseLen += int(n)

	if len(seq.Ops) > 0 {
		lastIdx := len(seq.Ops) - 1
		if last, ok := seq.Ops[lastIdx].(Delete); ok {
			seq.Ops[lastIdx] = Delete{N: last.N + n}
			return
		}
	}
	seq.Ops = append(seq.Ops, Delete{N: n})
}

func (seq *Sequence) Insert(s string) {
	n := utf8.RuneCountInString(s)
	if n == 0 {
		return
	}
	seq.TargetLen += n

	if len(seq.Ops) > 0 {
		lastIdx := len(seq.Ops) - 1
		switch lastOp := seq.Ops[lastIdx].(type) {
		case Insert:
			seq.Ops[lastIdx] = Insert{Str: lastOp.Str + s}
			return
		case Delete:
			if len(seq.Ops) >= 2 {
				preLastIdx := lastIdx - 1
				if preLast, ok := seq.Ops[preLastIdx].(Insert); ok {
					seq.Ops[preLastIdx] = Insert{Str: preLast.Str + s}
					return
				}
			}
			seq.Ops[lastIdx] = Insert{Str: s}
			seq.Ops = append(seq.Ops, lastOp)
			return
		}
	}
	seq.Ops = append(seq.Ops, Insert{Str: s})
}

func (seq *Sequence) Retain(n uint64) {
	if n == 0 {
		return
	}
	seq.BaseLen += int(n)
	seq.TargetLen += int(n)

	if len(seq.Ops) > 0 {
		lastIdx := len(seq.Ops) - 1
		if last, ok := seq.Ops[lastIdx].(Retain); ok {
			seq.Ops[lastIdx] = Retain{N: last.N + n}
			return
		}
	}
	seq.Ops = append(seq.Ops, Retain{N: n})
}

func (seq *Sequence) Apply(s string) (string, error) {
	if utf8.RuneCountInString(s) != seq.BaseLen {
		return "", ErrIncompatibleLengths
	}

	var result strings.Builder
	chars := []rune(s)
	pos := 0

	for _, op := range seq.Ops {
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
	if len(seq.Ops) == 0 {
		return true
	}
	if len(seq.Ops) == 1 {
		_, ok := seq.Ops[0].(Retain)
		return ok
	}
	return false
}

func (seq *Sequence) Compose(other *Sequence) (*Sequence, error) {
	if seq.TargetLen != other.BaseLen {
		return nil, ErrIncompatibleLengths
	}

	newOp := NewSequence()
	ops1 := cloneOps(seq.Ops)
	ops2 := cloneOps(other.Ops)
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
	if seq.BaseLen != other.BaseLen {
		return nil, nil, ErrIncompatibleLengths
	}

	aPrime := NewSequence()
	bPrime := NewSequence()

	ops1 := cloneOps(seq.Ops)
	ops2 := cloneOps(other.Ops)
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

	for _, op := range seq.Ops {
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

func (seq *Sequence) MarshalJSON() ([]byte, error) {
	var anyOps []interface{}
	for _, op := range seq.Ops {
		var anyOp interface{}
		switch val := op.(type) {
		case Retain:
			anyOp = val.N
		case Delete:
			anyOp = -int(val.N)
		case Insert:
			anyOp = val.Str
		}
		anyOps = append(anyOps, anyOp)
	}
	return json.Marshal(anyOps)
}

func (seq *Sequence) UnmarshalJSON(data []byte) error {
	var ops []interface{}
	if err := json.Unmarshal(data, &ops); err != nil {
		return err
	}
	for _, op := range ops {
		switch val := op.(type) {
		case int:
			if val > 0 {
				seq.Retain(uint64(val))
			}
			seq.Delete(uint64(-val))
		case float64:
			if val > 0 {
				seq.Retain(uint64(val))
			}
			seq.Delete(uint64(-val))
		case string:
			seq.Insert(val)
		default:
			return fmt.Errorf("invalid operation type: %T", val)
		}
	}
	return nil
}
