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

type BTNode interface {
	GetParent() *BTNode
	SetParent(*BTNode)
	GetSibblings() (*BTNode, *BTNode)
	SetLeftSibbling(*BTNode)
	SetRightSibbling(*BTNode)

	SetChild(int)
	AddChild(Key, *BTNode)

	Find(Key) (*BTNode, int)
	InsertAt(int, Key, Value) 
	IsAt(int,Key) bool
	Remove(int) Key
	IsLeaf()
	NumKeys() int 
	Merge(*BTNode)
	Split(*BTNode)
}

type BTNodeProvider interface {
	GetNewNode() *BTNode
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
	log.Printf("Add key=%v, value=%v", key, value)
	node, idx := find(b.root, key)

	if node == nil {
		log.Printf("Tree is empty, adding new leaf")
		node = makeNode(true, b.nodeSize, b.root)
		node.insertAt(idx, key, value)
		b.root.Keys[0] = key
		b.root.Child[1] = node
	} else if node.Leaf {
		log.Printf("Found a leaf node with keys %v, inserting on it at index %d", node, idx)
		nodeSize, e := node.insertAt(idx, key, value)

		if e != nil {
			fmt.Printf("Couldn't insert value in position %d", idx)
		}

		b.checkOverflow(node, nodeSize)
		// leaf does not exist, create a new one
	} else {
		log.Printf("Found node with keys %v empty position %d, creating new leaf node", node, idx)
		newLeaf := makeNode(true, b.nodeSize, node)
		newLeaf.insertAt(0, key, value)
		nodeSize := node.insertNode(idx, key, newLeaf)
		b.checkOverflow(node, nodeSize)
	}

	b.keys += 1
}

func (b *BTree) checkOverflow(node *Node, nodeSize int) {
	// if after write the nodeSize overflow the leaf node, split the node
	// and add it to it's parent
	it := node
	log.Println("Checking split")
	for nodeSize > b.nodeSize && it != nil {
		log.Printf("Splitting node %v\n", it)
		parent := it.Parent
		
		split, key := it.split(b.nodeSize)
		log.Printf("left: %v it: %v right: %v", it.Left, it, split)

		if it.Leaf {
			right := it.Right
			if right != nil {
				right.Left = split 
				split.Right = right
			}

			it.Right = split
			split.Left = it 
		}

		if parent == nil {
			newRoot := makeNode(false, b.nodeSize, nil)

			newRoot.Keys = append(newRoot.Keys, key)
			newRoot.Child = append(newRoot.Child, it)
			newRoot.Child = append(newRoot.Child, split)

			it.Parent = newRoot
			split.Parent = newRoot
			b.root = newRoot
			b.depth += 1
			break
		}


		log.Printf("Splited node %v with split key %s\n", split, key)

		idx := binarySearch(parent, key)
		log.Printf("Inserting split key at parent keys %v index %d\n", parent, idx)
		nodeSize = parent.insertNode(idx, key, split)
		it = parent
	}
}

func (b *BTree) Remove(key Key) (Value, error) {
	node, idx := find(b.root, key)

	if !node.isAt(key, idx) {
		log.Printf("Not found key %v at index %d in node %v", key, idx, node)
		return Value{}, fmt.Errorf("Key %v not found", key)
	}

	ret := node.remove(idx)
	b.checkUnderflow(node)
	b.keys-- 

	return ret, nil
}

func (b *BTree) checkUnderflow(node *Node) {
	if node == b.root {
		if !node.Leaf && len(node.Keys) == 0 {
			log.Printf("Root node is empty. Promoting %v to root", node.Child[0])
			b.root = node.Child[0]
			b.root.Parent = nil
			b.depth--
			return 
		}
	}
	
	parent := node.Parent 
	nodeIdx, left, right := node.GetSibblings()
	var key Key 
	
	if len(node.Keys) >= b.minKeys() {
		return 
	}

	if left != nil && left.Size() > b.minKeys() {
		key = left.Keys[len(left.Keys)-1]
		sepKeyIdx := nodeIdx - 1
		log.Printf("Rebalancing node %v with left %v", node, left)
		if node.Leaf {
			// 1. get the value from the rightmost key 
			value := left.Values[len(left.Values)-1]
			// 2. open space on the node to add the value 
			node.Keys = InsertAt(node.Keys, key, 0)
			node.Values = InsertAt(node.Values, value, 0)
			// 4. update the parent key 
			parent.Keys[sepKeyIdx] = key
			// shrink the left node 
			left.Keys, _ = Remove(left.Keys, len(left.Keys)-1) 
			left.Values, _ = Remove(left.Values, len(left.Values)-1)
		} else { 
			// get the rightmost node
			n := left.Child[len(left.Child)-1]
			node.Keys = InsertAt(node.Keys, parent.Keys[sepKeyIdx], 0)
			node.Child = InsertAt(node.Child, n, 0) 
			// promote the previous key to the parent note 
			parent.Keys[sepKeyIdx] = key
			// shrink the left node 
			left.Keys, _ = Remove(left.Keys, len(left.Keys)-1)
			left.Child, _ = Remove(left.Child, len(left.Child)-1)
			// update  parent and left/right relationship 
			n.Parent = node 
		}
	} else if right != nil && right.Size() > b.minKeys() {
		log.Printf("Rebalancing node %v with right %v", node, right)
		key := right.Keys[0]
		sepKeyIdx := nodeIdx
		if node.Leaf {
			// get the value from the leftmost key on the right node 
			value := right.Values[0]
			// oppend it on the node 
			node.Keys = append(node.Keys, key)
			node.Values = append(node.Values, value)
			// remove the key from the right node and shrink it  
			right.Keys, _ = Remove(right.Keys, 0)
			right.Values, _ = Remove(right.Values, 0)
			// update the parent sep key 
			newSepKey := right.Keys[0]
			parent.Keys[sepKeyIdx] = newSepKey
		} else {
			// get the children node from the leftmost position on the right node
			n := right.Child[0]
			sepKey := node.Parent.Keys[sepKeyIdx]
			// append it to the current node 
			node.Keys = append(node.Keys, sepKey)
			node.Child = append(node.Child, n)
			//remove key/children from the right node 
			right.Keys, _ = Remove(right.Keys, 0)
			right.Child, _ = Remove(right.Child, 0)
			// update the parent sep key 
			parent.Keys[sepKeyIdx] = key
			// update parent and left/right relationship 
			n.Parent = node
		}
	} else if left != nil && node.Size() + left.Size() <= b.nodeSize {
		log.Printf("Merging node %v with left node %v", node, left)
		sepKeyIdx := nodeIdx - 1
		sepKey := parent.Keys[sepKeyIdx]
		if node.Leaf {
			// copy values from the left node 
			node.Keys = append(left.Keys, node.Keys...)
			node.Values = append(left.Values, node.Values...)
			// remove separator key from the parent node 
			parent.Keys, _ = Remove(parent.Keys, sepKeyIdx)
			parent.Child, _ = Remove(parent.Child, sepKeyIdx)
			// fix the left right relationship
			ll := left.Left
			if ll != nil {
				ll.Right = node 
			}
			node.Left = ll
		} else {
			node.Keys = append([]Key{sepKey}, node.Keys...)
			node.Keys = append(left.Keys, node.Keys...)
			node.Child = append(left.Child, node.Child...)
			
			for _, children := range left.Child {
				children.Parent = node
			}
			 
			parent.Keys, _ = Remove(parent.Keys, sepKeyIdx)
			parent.Child, _ = Remove(parent.Child, sepKeyIdx)
		}

		log.Printf("Resulting node %v. Propagating to parent %v", node, parent)
		b.checkUnderflow(parent)
	} else if right != nil && node.Size() + right.Size() <= b.nodeSize {
		log.Printf("Merging node %v with right node %v", node, right)	
		sepKeyIdx := nodeIdx
		sepKey := parent.Keys[sepKeyIdx]
		
		if node.Leaf {
			// copy keys/values from right node 
			node.Keys = append(node.Keys, right.Keys...)
			node.Values = append(node.Values, right.Values...)
			// remove the key/child from the parent
			parent.Keys, _ = Remove(parent.Keys, sepKeyIdx)
			parent.Child, _ = Remove(parent.Child, sepKeyIdx+1)
			// fix right/left relationship

			rr := right.Right
			if rr != nil {
				rr.Left = node 
			}
			node.Right = rr

		} else {
			node.Keys = append(node.Keys, sepKey)
			node.Keys = append(node.Keys, right.Keys...)
			node.Child = append(node.Child, right.Child...)
			
			for _, children := range right.Child {
				children.Parent = node 
			}
			
			parent.Keys, _ = Remove(parent.Keys, sepKeyIdx)
			parent.Child, _ = Remove(parent.Child, sepKeyIdx+1 )
		}
		b.checkUnderflow(parent)
	}
}

func (b *BTree) Find(key Key) (Value, error) {

	node, idx := find(b.root, key)

	if !node.isAt(key, idx) {
		return Value{}, fmt.Errorf("Key %v not found", key)
	}
	return node.Values[idx], nil
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

