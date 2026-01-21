package ds

type Key interface {
	Compare (other *Key) int 
} 
type Value struct {
	v []byte
}

type Node struct {
	leaf bool
	keys []Key 
	child *[]Node 
	values *[]Value
}

type BTree struct {
	root Node
	depth int 
	keys int 
	nodeSize int
}


func NewBTree() *BTree {
	return &BTree{
		leaf: false,
		keys: make([]Key),
		nodes: make(*[]Node),
		values: nil 
	}
}

func (*BTree) Add(key Key, value []byte) {
}

func (*BTree) Remove(key Key) []byte {
}

func (*BTree) Find(key Key) []byte {
}

func find(node *Node, key *Key) (*Node, int){
	for l := 0, r := len(node->keys); l != r; {
		mid := (l + r)/2

		midKey := node.keys[mid]
		
		compare := key.compare() 
		if compare > 0 {
			l = mid
		} else if compare < 0 {
			r = mid
		} else if !node.leaf {
			return find(node.child, key)
		} else {
			return node, mid
		}
	}

	return node, 0
}

func merge(left *Node, right *Node) {}
func split (node *Node) {}
