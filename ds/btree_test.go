package ds 

import(
	"testing"
)


type TestKey struct {
	k int
}

func (t TestKey) compare (other Key) return {
	otherT := (TestKey) other

	if t.k == otherT.k {
		return 0
	} else if t.k > otherT.k {
		return 1
	} else {
		return -1
	}
}

func TestAdd(t *testing.T){
	bt := NewBTree()
	
	for i := range(1, 10) {

		bt.Add()
	}
}

