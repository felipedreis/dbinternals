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
	parent *Node
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

func makeNode(leaf bool, nodeSize int, parent *Node) *Node {
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
		parent: parent,
	}
}


func NewBTree(nodeSize int) *BTree {
	var rootNode = makeNode(false, nodeSize, nil)

	return &BTree{
		root: rootNode,
		depth: 1,
		keys: 0,
		nodeSize: nodeSize,
	}
}

func (b *BTree) Add(key Key, value []byte) {
	node, idx := find(b.root, key, true)

	if node == nil {
		// Tree is empty, add the first leaf bellow the root
		node = makeNode(true, b.nodeSize, b.root)
		node.insertAt(idx, key, value)
		b.root.child[0] = node 
	} else {
		// found an existing leaf, add the value at the right position
		nodeSize, error = node.insertAt(idx, key, value) 
		
		if error != nil {
			fmt.Printf("Couldn't insert value in position %d", idx)
		}

		// if after write the nodeSize overflow the leaf node, split the node 
		// and add it to it's parent  
		for ; nodeSize > b.nodeSize && node != nil ; {
			parent := node.parent

			right, key = node.split()
			
			// find the key position in the parent node 
			idx = find(parent, key, false)
			nodeSize = parent.insertNode(idx, key, right)
			node = parent
		}

		// TODO check if the root node has to split
	}
}

func (*BTree) Remove(key Key) []byte {
	return make([]byte, 1)
}

func (*BTree) Find(key Key) []byte {
	return make([]byte, 1)
}

func find(node *Node, key Key, descend boolean) (*Node, int) {
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

func (n *Node) split(nodeSize int) (*Node, Key) {
	right := makeNode(n.leaf, nodeSize, n.parent)
	
	mid := len(n.keys)/2

	if n.leaf {
		// copy mid keys and mid values
		right.keys[0:] = n.keys[mid:]
		right.values[0:] = n.values[mid:]

		for i := mid; i < len(n.keys); i += 1 {
			n.keys[i] = nil
			n.value[i] = nil 
		}

		
	} else {
		// copy last k keys and last k + 1 child nodes
		right.keys[0:] = n.keys[mid:]
		right.child[0:] = n.child[mid:]
		
		for i := mid; i < len(n.keys); i += 1 {
			n.keys[i] = nil
			n.child[i] = nil 
		}

		n.keys = n.keys[:mid]
		n.child = n.child[:mid]
	}

	return right, mid
}

func (n *Node) insertNode(index int, key Key, node *Node) int {
	// TODO implement me
	return 0
}
