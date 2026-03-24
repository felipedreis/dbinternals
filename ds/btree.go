package ds

import (
	"errors"
	"fmt"
	"log"
)

type Key interface {
	fmt.Stringer
	Compare(other Key) int
}
type Value struct {
	v []byte
}

type Node struct {
	leaf   bool
	parent *Node
	keys   []Key
	child  []*Node
	values []Value

	left  *Node
	right *Node
}

type NodeState int

const (
	NONE NodeState = iota
	REBALANCE_LEFT
	REBALANCE_RIGHT
	MERGE_LEFT
	MERGE_RIGHT
)

type BTree struct {
	root     *Node
	depth    int
	keys     int
	nodeSize int
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

func NewBTree(nodeSize int) *BTree {
	var rootNode = makeNode(true, nodeSize, nil)

	return &BTree{
		root:     rootNode,
		depth:    1,
		keys:     0,
		nodeSize: nodeSize,
	}
}

func (b *BTree) Add(key Key, value Value) {
	node, idx := find(b.root, key)

	if node == nil {
		log.Printf("Tree is empty, adding new leaf")
		node = makeNode(true, b.nodeSize, b.root)
		node.insertAt(idx, key, value)
		b.root.keys[0] = key
		b.root.child[1] = node
	} else if node.leaf {
		log.Printf("Found a leaf node with keys %v, inserting on it at index %d", node, idx)
		nodeSize, e := node.insertAt(idx, key, value)

		if e != nil {
			fmt.Printf("Couldn't insert value in position %d", idx)
		}

		b.propagateSplit(node, nodeSize)
		// leaf does not exist, create a new one
	} else {
		log.Printf("Found node with keys %v empty position %d, creating new leaf node", node, idx)
		newLeaf := makeNode(true, b.nodeSize, node)
		newLeaf.insertAt(0, key, value)
		nodeSize := node.insertNode(idx, key, newLeaf)
		b.propagateSplit(node, nodeSize)
	}

	b.keys += 1
}

func (b *BTree) propagateSplit(node *Node, nodeSize int) {
	// if after write the nodeSize overflow the leaf node, split the node
	// and add it to it's parent
	splitNode := node
	log.Println("Checking split")
	for nodeSize > b.nodeSize && node != nil {
		log.Printf("Splitting node %v\n", splitNode)
		parent := splitNode.parent

		if parent == nil {
			newRoot := makeNode(false, b.nodeSize, nil)

			right, key := splitNode.split(b.nodeSize)

			newRoot.keys = append(newRoot.keys, key)
			newRoot.child = append(newRoot.child, splitNode)
			newRoot.child = append(newRoot.child, right)

			splitNode.right = right
			right.left = splitNode
			splitNode.left = nil
			right.right = nil

			splitNode.parent = newRoot
			right.parent = newRoot
			b.root = newRoot
			b.depth += 1
			break
		}

		right, key := splitNode.split(b.nodeSize)

		log.Printf("Splited node %v with split key %s\n", right, key)

		idx := binarySearch(parent, key)
		log.Printf("Inserting split key at parent keys %v index %d\n", parent, idx)
		nodeSize = parent.insertNode(idx, key, right)
		splitNode = parent
	}
}

func (b *BTree) Remove(key Key) (Value, error) {
	node, idx := find(b.root, key)

	if !node.isAt(key, idx) {
		return Value{}, fmt.Errorf("Key %v not found", key)
	}

	ret := node.remove(idx)

	// early return, root node has no sibblings to merge/rebalance
	if node == b.root {
		return ret, nil
	}

	state := node.checkState(b.nodeSize)

	switch state {
	case REBALANCE_LEFT:
		b.merge(node.left, node)
	case REBALANCE_RIGHT:
		b.rebalance(node, node.right)
	case MERGE_LEFT:
		b.merge(node.left, node)
	case MERGE_RIGHT:
		b.merge(node, node.right)
	}

	return ret, nil
}

func (b *BTree) merge(left *Node, right *Node) error {
	if left.parent != right.parent {
		return errors.New("Can't merge non sibbling nodes")
	}

	left.keys = append(left.keys, right.keys...)
	left.values = append(left.values, right.values...)
	left.right = right.right

	parent := left.parent
	keysCount := len(parent.keys)

	idx := binarySearch(parent, right.keys[0])
	// TODO fix this shit
	copy(parent.keys[idx:], parent.keys[idx+1:]) // corner case if key is the last
	copy(parent.child[i:], parent.child[i+1:])
	parent.keys = parent.keys[:keysCount-1]
	parent.child = parent.child[:keysCount]

	return nil
}

func (b *BTree) rebalance(left *Node, right *Node) {}

func (n *Node) remove(idx int) Value {
	ret := n.values[idx]
	lastIdx := len(n.keys) - 1
	copy(n.keys[idx:], n.keys[idx+1:])
	copy(n.values[idx:], n.values[idx+1:])
	n.keys = n.keys[:lastIdx]
	n.values = n.values[:lastIdx]
	return ret
}

func (n *Node) checkState(nodeSize int) NodeState {

	var state NodeState = NONE
	var leftKeysCount int
	var rightKeysCount int
	keysCount := len(n.keys)

	switch {
	case n.right == nil && n.left != nil:
		leftKeysCount = len(n.left.keys)
		if keysCount+leftKeysCount < nodeSize {
			state = MERGE_LEFT
		} else {
			state = REBALANCE_LEFT
		}
	case n.left == nil && n.right != nil:
		rightKeysCount = len(n.right.keys)

		if keysCount+rightKeysCount < nodeSize {
			state = MERGE_RIGHT
		} else {
			state = REBALANCE_RIGHT
		}
	default:
		leftKeysCount = len(n.left.keys)
		rightKeysCount = len(n.right.keys)

		switch {
		case leftKeysCount+keysCount < nodeSize:
			state = MERGE_LEFT
		case rightKeysCount+keysCount < nodeSize:
			state = MERGE_RIGHT
		case leftKeysCount >= rightKeysCount:
			state = REBALANCE_LEFT
		default:
			state = REBALANCE_RIGHT
		}
	}

	return state
}

func (n *Node) isAt(key Key, index int) bool {
	keyAtIndex := n.keys[index]

	return n.leaf && keyAtIndex.Compare(key) == 0
}

func (b *BTree) Find(key Key) (Value, error) {

	node, idx := find(b.root, key)

	if !node.isAt(key, idx) {
		return Value{}, fmt.Errorf("Key %v not found", key)
	}
	return node.values[idx], nil
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

func find(node *Node, key Key) (*Node, int) {
	if node.leaf {
		return node, binarySearch(node, key)
	}

	idx := binarySearch(node, key)
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

func (b *BTree) Print() {
	fmt.Println("--- B+Tree Structure ---")
	if b.root != nil {
		printNode(b.root, 0)
	}
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
