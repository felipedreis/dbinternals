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


type BTree struct {
	root     *Node
	depth    int
	keys     int
	nodeSize int
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
		log.Printf("Not found key %v at index %d in node %v", key, idx, node)
		return Value{}, fmt.Errorf("Key %v not found", key)
	}

	ret := node.remove(idx)

	b.propagateUnderflow(node)

	return ret, nil
}

func (b *BTree) propagateUnderflow(node *Node) {
	if node == b.root {
		if !node.leaf && len(node.keys) == 0 {
			b.root = node.child[0]
			b.root.parent = nil
			b.depth--
			return 
		}
	}
	
	parent := node.parent 
	parentIdx, left, right := node.getSibblings()

	if len(node.keys) >= b.minKeys() {
		return 
	}

	if left != nil && left.size() > b.minKeys() {
		sepKeyIdx := parentIdx -1 
		key := left.keys[len(left.keys)-1]

		log.Printf("Rebalancing node %v with left %v", node, left)
		if node.leaf {
			// 1. get the value from the rightmost key 
			value := left.values[len(left.values)-1]
			// 2. open space on the node to add the value 
			node.keys = append(node.keys, nil)
			node.values = append(node.values, value)
			copy(node.keys[1:], node.keys[:len(node.keys)-2])
			copy(node.values[1:], node.values[:len(node.values)-2])
			// 3. copy the value into the leftmost position in the node 
			node.keys[0] = key
			node.values[0] = value
			// 4. update the parent key 
			parent.keys[sepKeyIdx] = key
			// shrink the left node 
			left.keys = left.keys[:len(left.keys)-1]
			left.values = left.values[:len(left.values)-1]
		} else { 
			// get the rightmost node
			n := left.child[len(left.child)-1]
			// open space on the current node to add the new key/child 
			node.keys = append(node.keys, nil)
			node.child = append(node.child, nil) 
			copy(node.keys[1:], node.keys[:len(node.keys)-2])
			copy(node.child[1:], node.child[:len(node.child)-2])
			// demote the previous parent key to the current node 
			node.keys[0] = parent.keys[sepKeyIdx]
			// promote the previous key to the parent note 
			parent.keys[sepKeyIdx] = key
			// add the new child 
			node.child[0] = n
			// shrink the left node 
			left.keys = left.keys[:len(left.keys)-1]
			left.child = left.child[:len(left.child)-1]
			// update  parent and left/right relationship 
			n.parent = node 
			node.child[1].left = n
			n.right = node.child[1]
			n.left = left.child[len(left.child)-1] 
			left.child[len(left.child)-1].right = n 
		}
	} else if right != nil && right.size() > b.minKeys() {
		log.Printf("Rebalancing node %v with right %v", node, right)
		sepKeyIdx := parentIdx 
		key := right.keys[0]
		if node.leaf {
			// get the value from the leftmost key on the right node 
			value := right.values[0]
			// oppend it on the node 
			node.keys = append(node.keys, key)
			node.values = append(node.values, value)
			// remove the key from the right node and shrink it  
			copy(right.keys[0:len(right.keys)-2], right.keys[1:])
			copy(right.values[0:len(right.values)-2], right.values[1:])
			right.keys = right.keys[:len(right.keys)-1]
			right.values = right.values[:len(right.values)-1]
			// update the parent sep key 
			newSepKey := right.keys[0]
			parent.keys[sepKeyIdx] = newSepKey
		} else {
			// get the children node from the leftmost position on the right node
			n := right.child[0]
			// append it to the current node 
			node.keys = append(node.keys, key)
			node.child = append(node.child, n)
			//remove key/children from the right node 
			copy(right.keys[0:len(right.keys)-1], right.keys[1:])
			copy(right.child[0:len(right.child)-1], right.child[1:])
			right.keys = right.keys[:len(right.keys)-1]
			right.child = right.child[:len(right.child)-1]
			// update the parent sep key 
			newSepKey := right.keys[0]
			parent.keys[sepKeyIdx] = newSepKey
			// update parent and left/right relationship 
			n.parent = node
			node.child[len(node.child)-2].right = n 
			n.left = node.child[len(node.child)-2]
			right.child[0].left = n
			n.right = right.child[0] 
		}
	} else if left != nil && node.size() + left.size() <= b.nodeSize {
		log.Printf("Merging node %v with left node %v", node, left)	
		if node.leaf {
		} else {
		} 
	} else if right != nil && node.size() + right.size() <= b.nodeSize {
		sepKeyIdx := parentIdx
		log.Printf("Merging node %v with left node %v", node, right)	
		if node.leaf {
			// copy keys from right node 
			node.keys = append(node.keys, right.keys...)
			// copy values from right 
			node.values = append(node.values, right.values...)
			// remove the key from the parent
			copy(parent.keys[sepKeyIdx:], parent.keys[sepKeyIdx+1:])
			parent.keys = parent.keys[:len(parent.keys)-1]
			// remove the node from the parent 
			copy(parent.child[sepKeyIdx+1:], parent.child[sepKeyIdx+2:])
			parent.child[len(parent.child)-1] = nil
			parent.child = parent.child[:len(parent.keys)-1]
			// fix right/left relationship
			node.left = right.left 
			right.left.right = node
		} else {
		}

		b.propagateUnderflow(parent)
	}
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
	copy(parent.child[idx:], parent.child[idx+1:])
	parent.keys = parent.keys[:keysCount-1]
	parent.child = parent.child[:keysCount]

	return nil
}

func (b *BTree) Find(key Key) (Value, error) {

	node, idx := find(b.root, key)

	if !node.isAt(key, idx) {
		return Value{}, fmt.Errorf("Key %v not found", key)
	}
	return node.values[idx], nil
}


func (b *BTree) Print() {
	fmt.Println("--- B+Tree Structure ---")
	if b.root != nil {
		printNode(b.root, 0)
	}
}

func (b BTree) minKeys() int {
	return (b.nodeSize + 1)/2
}

