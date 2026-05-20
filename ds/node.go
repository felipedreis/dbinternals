package ds

import (
	"errors"
	"fmt"
	"log"
)

type NodeState int

const (
	NONE NodeState = iota
	REBALANCE_LEFT
	REBALANCE_RIGHT
	MERGE_LEFT
	MERGE_RIGHT
)

type Node struct {
	leaf   bool
	parent *Node
	keys   []Key
	child  []*Node
	values []Value

	left  *Node
	right *Node
}

func makeNode(leaf bool, nodeSize int, parent *Node) *Node {
	var values []Value = nil
	var child []*Node = nil

	if leaf {
		values = make([]Value, 0, nodeSize)
	} else {
		child = make([]*Node, 0, nodeSize+1)
	}

	return &Node{
		leaf:   leaf,
		keys:   make([]Key, 0, nodeSize),
		child:  child,
		values: values,
		parent: parent,
	}
}

func printNode(n *Node, level int) {
	// Create indentation (2 spaces per level)
	indent := ""
	for range level {
		indent += "  "
	}

	// Print the node info
	fmt.Printf("%s %v\n", indent, n)

	// Recurse into children if not a leaf
	if !n.leaf {
		for _, child := range n.child {
			if child != nil {
				printNode(child, level+1)
			}
		}
	}
}

func binarySearch(node *Node, key Key) int {
	l := 0
	r := len(node.keys)

	for l != r {
		mid := (l + r) / 2

		compare := key.Compare(node.keys[mid])
		if compare > 0 {
			l = mid + 1
		} else if compare < 0 {
			r = mid
		} else {
			return mid
		}
	}

	return l
}

func (n *Node) getSibblings() (int, *Node, *Node) {
	var left *Node 
	var right *Node
	parent := n.parent
	
	if parent == nil || n.isEmpty() {
		return -1, nil, nil
	}
	
	nodeIdx := binarySearch(parent, n.keys[0])
	
	if parent.child[nodeIdx] != n {
		nodeIdx--
	} 

	if nodeIdx > 0 {
		left = parent.child[nodeIdx- 1]
	}
	if nodeIdx < len(parent.child) - 1 {
		right = parent.child[nodeIdx + 1]
	}

	return nodeIdx, left, right
}

func find(node *Node, key Key) (*Node, int) {
	log.Printf("looking for key %v at node %v", key, node)
	if node.leaf {
		idx := binarySearch(node, key) 
		log.Printf("Found key %v at Leaf node index %d", key, idx)
		return node, idx
	}

	idx := binarySearch(node, key)
	
	log.Printf("Descending at idx %d", idx) 
	return find(node.child[idx], key)
}

func (n *Node) insertAt(index int, key Key, val Value) (int, error) {
	if !n.leaf {
		return 0, errors.New("Only insert values at leaf nodes")
	}

	n.values = append(n.values, val)
	n.keys = append(n.keys, nil)

	copy(n.values[index+1:], n.values[index:])
	copy(n.keys[index+1:], n.keys[index:])

	n.values[index] = val
	n.keys[index] = key

	return len(n.keys), nil
}

func (n *Node) remove(idx int) Value {
	ret := n.values[idx]
	lastIdx := len(n.keys) - 1
	copy(n.keys[idx:], n.keys[idx+1:])
	copy(n.values[idx:], n.values[idx+1:])
	n.keys = n.keys[:lastIdx]
	n.values = n.values[:lastIdx]
	return ret
}

func (n *Node) split(nodeSize int) (*Node, Key) {
	right := makeNode(n.leaf, nodeSize, n.parent)

	mid := len(n.keys) / 2
	midKey := n.keys[mid]

	if n.leaf {
		// copy mid keys and mid values
		right.keys = append(right.keys, n.keys[mid:]...)
		right.values = append(right.values, n.values[mid:]...)
		n.keys = n.keys[:mid]
		n.values = n.values[:mid]
	} else {
		// copy last k keys and last k + 1 child nodes
		// the mid key goes up to the next level
		right.keys = append(right.keys, n.keys[mid+1:]...)
		right.child = append(right.child, n.child[mid+1:]...)
		n.keys = n.keys[:mid]
		n.child = n.child[:mid+1]

		for _, children := range right.child {
			children.parent = right
		}

		n.child[len(n.child)-1].right = nil
		right.child[0].left = nil
	}

	return right, midKey
}

func (n *Node) insertNode(index int, key Key, node *Node) int {
	n.keys = append(n.keys, nil)
	n.child = append(n.child, nil)
	copy(n.keys[index+1:], n.keys[index:])
	copy(n.child[index+2:], n.child[index+1:])
	n.keys[index] = key
	n.child[index+1] = node

	n.child[index].right = node
	node.left = n.child[index]

	if index+2 < len(n.child) {
		n.child[index+2].left = node
		node.right = n.child[index+2]
	} else {
		node.right = nil
	}

	return len(n.keys)
}

func (n *Node) isAt(key Key, index int) bool {
	if n.isEmpty() || index >= len(n.keys) {
		return false
	}

	keyAtIndex := n.keys[index]

	return n.leaf && keyAtIndex.Compare(key) == 0
}

func (n Node) isEmpty() bool {
	return len(n.keys) == 0
}

func (n Node) size() int {
	return len(n.keys)
}

func (n Node) String() string {
	// Build a string of the keys in this node
	nodeType := "Internal"
	if n.leaf {
		nodeType = "Leaf"
	} else if n.parent == nil {
		nodeType = "Root"
	}

	return fmt.Sprintf("%s: %v", nodeType, n.keys)
}

