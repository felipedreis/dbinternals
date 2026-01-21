package ds 

import(
	"testing"
	"encoding/binary"
)


type TestKey struct {
	k int
}

func (t TestKey) Compare (other Key) int {
	otherT := other.(TestKey)

	if t.k == otherT.k {
		return 0
	} else if t.k > otherT.k {
		return 1
	} else {
		return -1
	}
}

func TestAdd(t *testing.T){
	bt := NewBTree(3)
	
	for i := 0; i < 10; i += 1 {
		testKey := TestKey{
			k: i,
		}
		v := make([]byte, 8)
		binary.AppendVarint(v, int64(i))

		bt.Add(testKey, v)
	}
}

