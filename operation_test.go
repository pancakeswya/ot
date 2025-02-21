package ot

import (
	"testing"
	"unicode/utf8"
	"reflect"
)

func TestLengths(t *testing.T) {
	o := NewSequence()
	if o.BaseLen != 0 {
		t.Fatal()
	}
	if o.TargetLen != 0 {
		t.Fatal()
	}
	o.Retain(5)
	if o.BaseLen != 5 {
		t.Fatal()
	}
	if o.TargetLen != 5 {
		t.Fatal()
	}
	o.Insert("abc")
	if o.BaseLen != 5 {
		t.Fatal()
	}
	if o.TargetLen != 8 {
		t.Fatal()
	}
	o.Retain(2)
	if o.BaseLen != 7 {
		t.Fatal()
	}
	if o.TargetLen != 10 {
		t.Fatal()
	}
	o.Delete(2)
	if o.BaseLen != 9 {
		t.Fatal()
	}
	if o.TargetLen != 10 {
		t.Fatal()
	}
}

func TestSequence(t *testing.T) {
	o := NewSequence()
	o.Retain(5)
	o.Retain(0)
	o.Insert("lorem")
	o.Insert("")
	o.Delete(3)
	o.Delete(0)
	if len(o.Ops) != 3 {
		t.Fatal()
	}
}

func TestApply(t *testing.T) {
	for i := 0; i < 1000; i++ {
		s := genString(50)
		o := genSequence(s)

		result, err := o.Apply(s)
		if err != nil {
			t.Fatal(err)
		}
		if utf8.RuneCountInString(s) != o.BaseLen {
			t.Fatal()
		}
		if utf8.RuneCountInString(result) != o.TargetLen {
			t.Fatal()
		}
	}
}

func TestIsNoop(t *testing.T) {
	o := NewSequence()
	if !o.IsNoop() {
		t.Fatal()
	}
	o.Retain(5)
	if !o.IsNoop() {
		t.Fatal()
	}
	o.Insert("lorem")
	if o.IsNoop() {
		t.Fatal()
	}
}

func TestEmptyOps(t *testing.T) {
	o := NewSequence()
	o.Retain(0)
	o.Insert("")
	o.Delete(0)
	if len(o.Ops) != 0 {
		t.Fatal()
	}
}

func TestCompose(t *testing.T) {
	for i := 0; i < 1000; i++ {
		s := genString(20)
		a := genSequence(s)
		afterA, err := a.Apply(s)
		if err != nil {
			t.Fatal(err)
		}
		if a.TargetLen != utf8.RuneCountInString(afterA) {
			t.Fatal()
		}
		b := genSequence(afterA)
		afterB, err := b.Apply(afterA)
		if err != nil {
			t.Fatal(err)
		}
		if b.TargetLen != utf8.RuneCountInString(afterB) {
			t.Fatal()
		}
		ab, err := a.Compose(b)
		if err != nil {
			t.Fatal(err)
		}
		if ab.TargetLen != b.TargetLen {
			t.Fatal()
		}
		afterAB, err := ab.Apply(s)
		if err != nil {
			t.Fatal(err)
		}
		if afterB != afterAB {
			t.Fatal()
		}
	}
}

func TestEq(t *testing.T) {
	o1 := NewSequence()
	o1.Delete(1)
	o1.Insert("lo")
	o1.Retain(2)
	o1.Retain(3)
	o2 := NewSequence()
	o2.Delete(1)
	o2.Insert("l")
	o2.Insert("o")
	o2.Retain(5)
	if !reflect.DeepEqual(o1, o2) {
		t.Fatal()
	}
	o1.Delete(1)
	o2.Retain(1)
	if reflect.DeepEqual(o1, o2) {
		t.Fatal()
	}
}

func TestOpsMerging(t *testing.T) {
	o := NewSequence()
	if len(o.Ops) != 0 {
		t.Fatal()
	}
	o.Retain(2)
	if len(o.Ops) != 1 {
		t.Fatal()
	}
	if !reflect.DeepEqual(o.Ops[len(o.Ops)-1], Retain{N: 2}) {
		t.Fatal()
	}
	o.Retain(3)
	if len(o.Ops) != 1 {
		t.Fatal()
	}
	if !reflect.DeepEqual(o.Ops[len(o.Ops)-1], Retain{N: 5}) {
		t.Fatal()
	}
	o.Insert("abc")
	if len(o.Ops) != 2 {
		t.Fatal()
	}
	if !reflect.DeepEqual(o.Ops[len(o.Ops)-1], Insert{Str: "abc"}) {
		t.Fatal()
	}
	o.Insert("xyz")
	if len(o.Ops) != 2 {
		t.Fatal()
	}
	if !reflect.DeepEqual(o.Ops[len(o.Ops)-1], Insert{Str: "abcxyz"}) {
		t.Fatal()
	}
	o.Delete(1)
	if len(o.Ops) != 3 {
		t.Fatal()
	}
	if !reflect.DeepEqual(o.Ops[len(o.Ops)-1], Delete{N: 1}) {
		t.Fatal()
	}
	o.Delete(1)
	if len(o.Ops) != 3 {
		t.Fatal()
	}
	if !reflect.DeepEqual(o.Ops[len(o.Ops)-1], Delete{N: 2}) {
		t.Fatal()
	}
}

func TestTransform(t *testing.T) {
	for i := 0; i < 1000; i++ {
		s := genString(20)
		a := genSequence(s)
		b := genSequence(s)

		aPrime, bPrime, err := a.Transform(b)
		if err != nil {
			t.Fatal(err)
		}
		abPrime, err := a.Compose(bPrime)
		if err != nil {
			t.Fatal(err)
		}
		baPrime, err := b.Compose(aPrime)
		if err != nil {
			t.Fatal(err)
		}
		afterABPrime, err := abPrime.Apply(s)
		if err != nil {
			t.Fatal(err)
		}
		afterBAPrime, err := baPrime.Apply(s)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(abPrime, baPrime) {
			t.Fatal()
		}
		if !reflect.DeepEqual(afterABPrime, afterBAPrime) {
			t.Fatal()
		}
	}
}

func TestInvert(t *testing.T) {
	for i := 0; i < 1000; i++ {
		s := genString(50)
		o := genSequence(s)
		p := o.Invert(s)
		if o.BaseLen != p.TargetLen {
			t.Fatal()
		}
		if o.TargetLen != p.BaseLen {
			t.Fatal()
		}
		afterO, err := o.Apply(s)
		if err != nil {
			t.Fatal(err)
		}
		afterInverse, err := p.Apply(afterO)
		if err != nil {
			t.Fatal(err)
		}
		if afterInverse != s {
			t.Fatal()
		}
	}
}
