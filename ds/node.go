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
	NodeId uint64
	
	Leaf   bool
	Parent *Node
	Keys   []Key
	Child  []*Node
	Values []Value

	Left  *Node
	Right *Node
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
		Leaf:   leaf,
		Keys:   make([]Key, 0, nodeSize),
		Child:  child,
		Values: values,
		Parent: parent,
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
	if !n.Leaf {
		for _, child := range n.Child {
			if child != nil {
				printNode(child, level+1)
			}
		}
	}
}


func (n *Node) GetSibblings() (int, *Node, *Node) {
	var left *Node 
	var right *Node
	parent := n.Parent
	var nodeIdx int 

	if parent == nil || n.IsEmpty() {
		return -1, nil, nil
	}
	
	
	for i, children := range parent.Child {
		if children == n {
			nodeIdx = i 
			break
		}	
	}

	if nodeIdx > 0 {
		left = parent.Child[nodeIdx- 1]
	}
	if nodeIdx < len(parent.Child) - 1 {
		right = parent.Child[nodeIdx + 1]
	}

	return nodeIdx, left, right
}

func find(node *Node, key Key) (*Node, int) {
	log.Printf("looking for key %v at node %v", key, node)
	if node.Leaf {
		idx := binarySearch(node, key) 
		log.Printf("Found key %v at Leaf node index %d", key, idx)
		return node, idx
	}

	idx := upperBound(node, key)
	

	log.Printf("Descending at idx %d", idx) 
	return find(node.Child[idx], key)
}

func binarySearch(node *Node, key Key) int {
	l := 0
	r := len(node.Keys)

	for l != r {
		mid := (l + r) / 2

		compare := key.Compare(node.Keys[mid])
		switch {
		case compare > 0: 
			l = mid + 1
		case compare < 0:
			r = mid
		default: 
			return mid
		}
	}

	return l
}

func upperBound(node *Node, key Key) int {
	l := 0
	r := len(node.Keys)

	for l < r {
		mid := (l + r) / 2

		compare := key.Compare(node.Keys[mid])
		if compare >= 0 { 
			l = mid + 1
		} else {
			r = mid
		}
	}

	return l
}

func (n *Node) insertAt(index int, key Key, val Value) (int, error) {
	if !n.Leaf {
		return 0, errors.New("Only insert values at leaf nodes")
	}
	
	n.Keys = InsertAt(n.Keys, key, index)
	n.Values = InsertAt(n.Values, val, index)

	return len(n.Keys), nil
}

func (n *Node) remove(idx int) Value {
	ret := n.Values[idx]
	lastIdx := len(n.Keys) - 1
	copy(n.Keys[idx:], n.Keys[idx+1:])
	copy(n.Values[idx:], n.Values[idx+1:])
	n.Keys = n.Keys[:lastIdx]
	n.Values = n.Values[:lastIdx]
	return ret
}

func (n *Node) split(nodeSize int) (*Node, Key) {
	right := makeNode(n.Leaf, nodeSize, n.Parent)

	mid := len(n.Keys) / 2
	midKey := n.Keys[mid]

	if n.Leaf {
		// copy mid keys and mid values
		right.Keys = append(right.Keys, n.Keys[mid:]...)
		right.Values = append(right.Values, n.Values[mid:]...)
		n.Keys = n.Keys[:mid]
		n.Values = n.Values[:mid]
	} else {
		// copy last k keys and last k + 1 child nodes
		// the mid key goes up to the next level
		right.Keys = append(right.Keys, n.Keys[mid+1:]...)
		right.Child = append(right.Child, n.Child[mid+1:]...)
		n.Keys = n.Keys[:mid]
		n.Child = n.Child[:mid+1]

		for _, children := range right.Child {
			children.Parent = right
		}
	}

	return right, midKey
}

func (n *Node) insertNode(index int, key Key, node *Node) int {
	n.Keys = InsertAt(n.Keys, key, index)
	n.Child = InsertAt(n.Child, node, index+1)
	return len(n.Keys)
}

func (n *Node) isAt(key Key, index int) bool {
	if n.IsEmpty() || index >= len(n.Keys) {
		return false
	}

	keyAtIndex := n.Keys[index]

	return n.Leaf && keyAtIndex.Compare(key) == 0
}

func (n Node) IsEmpty() bool {
	return len(n.Keys) == 0
}

func (n Node) IsRoot() bool {
	return n.Parent == nil
}

func (n Node) IsLeaf() bool {
	return n.Leaf
}

func (n Node) Size() int {
	return len(n.Keys)
}

func (n Node) String() string {
	// Build a string of the keys in this node
	nodeType := "Internal"
	if n.Leaf {
		nodeType = "Leaf"
	} else if n.Parent == nil {
		nodeType = "Root"
	}

	return fmt.Sprintf("%s: %v", nodeType, n.Keys)
}

