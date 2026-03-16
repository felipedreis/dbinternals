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
	values []*Value
}

type BTree struct {
	root     *Node
	depth    int
	keys     int
	nodeSize int
}

func makeNode(leaf bool, nodeSize int, parent *Node) *Node {
	var values []*Value = nil
	var child []*Node = nil

	if leaf {
		values = make([]*Value, 0, nodeSize)
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

func (b *BTree) Add(key Key, value *Value) {
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

func (*BTree) Remove(key Key) []byte {
	return make([]byte, 1)
}

func (*BTree) Find(key Key) Value {
	return Value{v: nil}
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
			return mid + 1
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

func (n *Node) insertAt(index int, key Key, val *Value) (int, error) {
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
