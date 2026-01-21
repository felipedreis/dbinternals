package ds

import (
	"errors"
)

type Key interface {
	Compare (other Key) int 
} 
type Value struct {
	v []byte
}

type Node struct {
	leaf bool
	keys []Key 
	child []*Node 
	values []Value
}

type BTree struct {
	root *Node
	depth int 
	keys int 
	nodeSize int
}

func makeNode(leaf bool, nodeSize int) *Node {
	var values []Value = nil
	var child []*Node = nil 

	if leaf {
		values = make([]Value, nodeSize) 
	} else {
		child = make([]*Node, nodeSize + 1)
	}


	return &Node{
		leaf: leaf,
		keys: make([]Key, nodeSize),
		child: child, 
		values: values,
	}
}


func NewBTree(nodeSize int) *BTree {
	var rootNode = makeNode(false, nodeSize)

	return &BTree{
		root: rootNode,
		depth: 1,
		keys: 0,
		nodeSize: nodeSize,
	}
}

func (*BTree) Add(key Key, value []byte) {
}

func (*BTree) Remove(key Key) []byte {
	return make([]byte, 1)
}

func (*BTree) Find(key Key) []byte {
	return make([]byte, 1)
}

func find(node *Node, key Key) (*Node, int) {
	l := 0
	r := len(node.keys)
	for ; l != r; {
		mid := (l + r)/2

		compare := key.Compare(node.keys[mid]) 
		if compare > 0 {
			l = mid + 1
		} else if compare < 0 {
			r = mid
		} else {
			if !node.leaf {
				return find(node.child[mid + 1], key)
			}

			return node, mid
		}
	}
	
	if !node.leaf {
		children := node.child[l]
		return find(children, key)
	} else {
		return node, l
	}

	return node, 0
}

func (n *Node) insertAt(index int, key Key, val Value) (int, error) {
	if !n.leaf {
		return 0, errors.New("Only insert values at leaf nodes")
	}

	n.values = append(n.values, Value{})
	n.keys = append(n.keys, nil)

	copy(n.values[index+1:], n.values[index:])
	copy(n.keys[index+1:], n.keys[index:])
	
	n.values[index] = val
	n.keys[index] = key
	
	return len(n.keys), nil
}

func merge(left *Node, right *Node) {}
func split (node *Node) {}
