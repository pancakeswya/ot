package ot

type Operation interface {
	isOperation()
}

type (
	Delete struct {
		N uint64
	}
	Retain struct {
		N uint64
	}
	Insert struct {
		Str string
	}
)

func (Delete) isOperation() {}
func (Retain) isOperation() {}
func (Insert) isOperation() {}
