//go:build wasm

package main

import (
	"syscall/js"
	"github.com/pancakeswya/ot"
)

// Registry to store sequences
var sequences = make(map[string]*ot.Sequence)

func main() {
	c := make(chan struct{})

	// Register constructors
	js.Global().Set("NewSequence", js.FuncOf(newSequence))
	js.Global().Set("NewDelete", js.FuncOf(newDelete))
	js.Global().Set("NewRetain", js.FuncOf(newRetain))
	js.Global().Set("NewInsert", js.FuncOf(newInsert))

	<-c
}

func newSequence(this js.Value, args []js.Value) interface{} {
	seq := ot.NewSequence()
	id := js.ValueOf(len(sequences)).String()
	sequences[id] = seq
	
	return js.ValueOf(map[string]interface{}{
		"ops": opsToJS(seq.Ops),
		"baseLen": seq.BaseLen,
		"targetLen": seq.TargetLen,
		"delete": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) > 0 {
				seq.Delete(uint64(args[0].Int()))
			}
			return nil
		}),
		"insert": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) > 0 {
				seq.Insert(args[0].String())
			}
			return nil
		}),
		"retain": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) > 0 {
				seq.Retain(uint64(args[0].Int()))
			}
			return nil
		}),
		"apply": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) > 0 {
				result, err := seq.Apply(args[0].String())
				if err != nil {
					return map[string]interface{}{
						"error": err.Error(),
					}
				}
				return result
			}
			return nil
		}),
		"isNoop": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			return seq.IsNoop()
		}),
		"compose": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) > 0 {
				otherID := args[0].Get("_sequenceID").String()
				other, exists := sequences[otherID]
				if !exists {
					return map[string]interface{}{
						"error": "invalid sequence object",
					}
				}
				
				result, err := seq.Compose(other)
				if err != nil {
					return map[string]interface{}{
						"error": err.Error(),
					}
				}
				return newSequenceValue(result)
			}
			return nil
		}),
		"transform": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) > 0 {
				otherID := args[0].Get("_sequenceID").String()
				other, exists := sequences[otherID]
				if !exists {
					return map[string]interface{}{
						"error": "invalid sequence object",
					}
				}
				
				aPrime, bPrime, err := seq.Transform(other)
				if err != nil {
					return map[string]interface{}{
						"error": err.Error(),
					}
				}
				return map[string]interface{}{
					"aPrime": newSequenceValue(aPrime),
					"bPrime": newSequenceValue(bPrime),
				}
			}
			return nil
		}),
		"invert": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) > 0 {
				inverse := seq.Invert(args[0].String())
				return newSequenceValue(inverse)
			}
			return nil
		}),
		"_sequenceID": id,
	})
}

func opsToJS(ops []ot.Operation) interface{} {
	result := make([]interface{}, len(ops))
	for i, op := range ops {
		switch v := op.(type) {
		case ot.Delete:
			result[i] = map[string]interface{}{
				"type": "delete",
				"n": v.N,
			}
		case ot.Retain:
			result[i] = map[string]interface{}{
				"type": "retain",
				"n": v.N,
			}
		case ot.Insert:
			result[i] = map[string]interface{}{
				"type": "insert",
				"str": v.Str,
			}
		}
	}
	return result
}

func newDelete(this js.Value, args []js.Value) interface{} {
	if len(args) > 0 {
		return js.ValueOf(map[string]interface{}{
			"type": "delete",
			"n": args[0].Int(),
		})
	}
	return nil
}

func newRetain(this js.Value, args []js.Value) interface{} {
	if len(args) > 0 {
		return js.ValueOf(map[string]interface{}{
			"type": "retain",
			"n": args[0].Int(),
		})
	}
	return nil
}

func newInsert(this js.Value, args []js.Value) interface{} {
	if len(args) > 0 {
		return js.ValueOf(map[string]interface{}{
			"type": "insert",
			"str": args[0].String(),
		})
	}
	return nil
}

func newSequenceValue(seq *ot.Sequence) js.Value {
	id := js.ValueOf(len(sequences)).String()
	sequences[id] = seq
	
	return js.ValueOf(map[string]interface{}{
		"ops": opsToJS(seq.Ops),
		"baseLen": seq.BaseLen,
		"targetLen": seq.TargetLen,
		"_sequenceID": id,
	})
}
