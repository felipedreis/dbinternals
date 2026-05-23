package ds

import (
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
			log.Printf("Root node is empty. Promoting %v to root", node.child[0])
			b.root = node.child[0]
			b.root.parent = nil
			b.depth--
			return 
		}
	}
	
	parent := node.parent 
	nodeIdx, left, right := node.getSibblings()
	var key Key 
	
	if len(node.keys) >= b.minKeys() {
		return 
	}

	if left != nil && left.size() > b.minKeys() {
		key = left.keys[len(left.keys)-1]
		sepKeyIdx := nodeIdx - 1
		log.Printf("Rebalancing node %v with left %v", node, left)
		if node.leaf {
			// 1. get the value from the rightmost key 
			value := left.values[len(left.values)-1]
			// 2. open space on the node to add the value 
			node.keys = InsertAt(node.keys, key, 0)
			node.values = InsertAt(node.values, value, 0)
			// 4. update the parent key 
			parent.keys[sepKeyIdx] = key
			// shrink the left node 
			left.keys, _ = Remove(left.keys, len(left.keys)-1) 
			left.values, _ = Remove(left.values, len(left.values)-1)
		} else { 
			// get the rightmost node
			n := left.child[len(left.child)-1]
			node.keys = InsertAt(node.keys, parent.keys[sepKeyIdx], 0)
			node.child = InsertAt(node.child, n, 0) 
			// promote the previous key to the parent note 
			parent.keys[sepKeyIdx] = key
			// shrink the left node 
			left.keys, _ = Remove(left.keys, len(left.keys)-1)
			left.child, _ = Remove(left.child, len(left.child)-1)
			// update  parent and left/right relationship 
			n.parent = node 
		}
	} else if right != nil && right.size() > b.minKeys() {
		log.Printf("Rebalancing node %v with right %v", node, right)
		key := right.keys[0]
		sepKeyIdx := nodeIdx
		if node.leaf {
			// get the value from the leftmost key on the right node 
			value := right.values[0]
			// oppend it on the node 
			node.keys = append(node.keys, key)
			node.values = append(node.values, value)
			// remove the key from the right node and shrink it  
			right.keys, _ = Remove(right.keys, 0)
			right.values, _ = Remove(right.values, 0)
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
			right.keys, _ = Remove(right.keys, 0)
			right.child, _ = Remove(right.child, 0)
			// update the parent sep key 
			newSepKey := right.keys[0]
			parent.keys[sepKeyIdx] = newSepKey
			// update parent and left/right relationship 
		}
	} else if left != nil && node.size() + left.size() <= b.nodeSize {
		log.Printf("Merging node %v with left node %v", node, left)
		sepKeyIdx := nodeIdx - 1
		sepKey := parent.keys[sepKeyIdx]
		if node.leaf {
			// copy values from the left node 
			node.keys = append(left.keys, node.keys...)
			node.values = append(left.values, node.values...)
			// remove separator key from the parent node 
			parent.keys, _ = Remove(parent.keys, sepKeyIdx)
			parent.child, _ = Remove(parent.child, sepKeyIdx)
			// fix the left right relationship
			node.left = left.left 
			if left.left != nil {
				left.left.right = node 
			}
		} else {
			node.keys = append([]Key{sepKey}, node.keys...)
			node.keys = append(left.keys, node.keys...)
			node.child = append(left.child, node.child...)
			
			for _, children := range left.child {
				children.parent = node
			}
			 
			parent.keys, _ = Remove(parent.keys, sepKeyIdx)
			parent.child, _ = Remove(parent.child, sepKeyIdx+1)
		}

		log.Printf("Resulting node %v. Propagating to parent %v", node, parent)
		b.propagateUnderflow(parent)
	} else if right != nil && node.size() + right.size() <= b.nodeSize {
		log.Printf("Merging node %v with right node %v", node, right)	
		sepKeyIdx := nodeIdx
		sepKey := parent.keys[sepKeyIdx]
		
		if node.leaf {
			// copy keys/values from right node 
			node.keys = append(node.keys, right.keys...)
			node.values = append(node.values, right.values...)
			// remove the key/child from the parent
			parent.keys, _ = Remove(parent.keys, sepKeyIdx)
			parent.child, _ = Remove(parent.child, sepKeyIdx+1)
			// fix right/left relationship
			node.right = right.right 
			if right.right != nil {
				right.right.left = node
			}
		} else {
			node.keys = append(node.keys, sepKey)
			node.keys = append(node.keys, right.keys...)
			node.child = append(node.child, right.child...)
			
			for _, children := range right.child {
				children.parent = node 
			}
			
			parent.keys, _ = Remove(parent.keys, sepKeyIdx)
			parent.child, _ = Remove(parent.child, sepKeyIdx+1 )
		}
		b.propagateUnderflow(parent)
	}
}


func InsertAt[T any](slice []T, value T, index int) []T{
	if index >= len(slice) {
		panic(fmt.Sprintf("Trying to insert at index %d outside of the slice %v", index, slice))
	}

	slice = append(slice, value)
	copy(slice[index+1:], slice[index:])
	slice[index] = value 
	return slice 
} 

func Remove[T any](slice []T, index int) ([]T, T) {
	if index >= len(slice) { 
		panic(fmt.Sprintf("Trying to remove at index %d outside of the slice %v", index, slice))
	}

	ret := slice[index]
	copy(slice[index:], slice[index+1:])
	slice = slice[:len(slice)-1]
	return slice, ret
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

