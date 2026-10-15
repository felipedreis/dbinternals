package ds

import (
	"fmt"
	"log"
	"slices"
)

type MasterPage interface {
	GetRootNode() NodeId
	SetRootNode(NodeId) 
}

type PagedBTree struct {
	masterPage MasterPage
	nodeProvider BTNodeProvider
}

func (t *PagedBTree) Add(key Key, value Value) {
	idx, splitList := t.find(key) 
	splitList[0].InsertValueAt(idx, key,  value) 
	t.checkOverflow(splitList)
}

func (t *PagedBTree) checkOverflow(splitList []BTNode) {
	it := len(splitList) - 1
	node := splitList[it] 

	for it < len(splitList) && node.IsOverflow() {
		split := t.nodeProvider.GetNewNode()
		splitKey := node.Split(split)
		
		if node.IsLeaf() {
			_, rightId := node.GetSibblings()
			
			if rightId != NoPage {
				rightNode := t.nodeProvider.GetNode(rightId)
				rightNode.SetLeftSibbling(split.GetId())
				split.SetRightSibbling(rightNode.GetId())
				t.nodeProvider.Release(rightNode.GetId(), true)
			}

			node.SetRightSibbling(split.GetId())
			split.SetLeftSibbling(node.GetId())
		}

		if it == len(splitList) - 1{
			newRoot := t.nodeProvider.GetNewNode()
			newRoot.Init(splitKey, node.GetId(), split.GetId())
			t.masterPage.SetRootNode(newRoot.GetId())
			t.nodeProvider.Release(newRoot.GetId(), true)
			break
		}
		
		
		parent := splitList[it + 1]
		
		idx, _ := parent.Search(splitKey)
		parent.InsertNodeAt(idx, splitKey, split.GetId())

		t.nodeProvider.Release(node.GetId(), true)
		t.nodeProvider.Release(split.GetId(), true)
		node = parent
		it++
	} 

	/// parent didn't split but still dirty 
	dirty := true 

	// release the remaining nodes
	for it >= 0 {
		t.nodeProvider.Release(splitList[it].GetId(), dirty)
		dirty = false 
		it--
	}
}

func (t *PagedBTree) Remove(key Key) (Value, error)  {
	idx, underflowList := t.find(key)
	defer t.checkUnderflow(underflowList)

	if !underflowList[0].IsAt(idx, key) {
		return Value{}, fmt.Errorf("Key %v not found", key)
	}
	
	value := underflowList[0].Remove(idx)
	return value, nil
}
 
func (t *PagedBTree) checkUnderflow(underflowList []BTNode) {
	lastParentChanged := false 
	var it int
	var node BTNode
	var lastReleased int

	for it, node = range underflowList {
		if !node.IsUnderflow() {
			t.nodeProvider.Release(node.GetId(), lastParentChanged)
			lastParentChanged = false
			lastReleased = it
			break 
		}
		
		// root node has zero keys, need to shrink 
		if it == len(underflowList) - 1 {
			if node.NumKeys() == 0 && !node.IsLeaf() {
				_, childNodeId := node.GetNodeAt(0)
				t.masterPage.SetRootNode(childNodeId)
				t.nodeProvider.Release(node.GetId(), true)
			} else {
				t.nodeProvider.Release(node.GetId(), lastParentChanged)
				lastParentChanged = false
			}
			
			lastReleased = it
			break
		}

		parent := underflowList[it + 1] 
		
		nodeIdx, leftNodeId, rightNodeId := getSibblings(node, parent)
		var leftNode BTNode
		var rightNode BTNode

		if leftNodeId != NoPage {
			leftNode = t.nodeProvider.GetNode(leftNodeId)
		}

		if rightNodeId != NoPage {
			rightNode = t.nodeProvider.GetNode(rightNodeId)
		}

		if leftNode != nil && !leftNode.IsUnderflow() {
			log.Printf("Rebalancing node %d with left %d", node.GetId(), leftNode.GetId())
			sepKeyIdx := nodeIdx - 1
			if node.IsLeaf() {
				key, value := leftNode.GetValueAt(leftNode.NumKeys() - 1)
				node.InsertValueAt(0, key, value)
				leftNode.Remove(leftNode.NumKeys() - 1)
				
				parent.SetKeyAt(sepKeyIdx, key)
				lastParentChanged = true 
				
				t.nodeProvider.Release(node.GetId(), true)
				t.nodeProvider.Release(leftNode.GetId(), true)
			} else {
				key, childNodeId := leftNode.GetNodeAt(leftNode.NumKeys() - 1)
				leftNode.Remove(leftNode.NumKeys() - 1)
				childNode := t.nodeProvider.GetNode(childNodeId)

				sepKey := parent.GetKeyAt(sepKeyIdx)
				parent.SetKeyAt(sepKeyIdx, key)
				lastParentChanged = true
				
				node.InsertNodeAt(0, sepKey, childNodeId)
				childNode.SetParent(node.GetId())

				t.nodeProvider.Release(node.GetId(), true)
				t.nodeProvider.Release(leftNode.GetId(), true)
				t.nodeProvider.Release(childNode.GetId(), true)
			}

			if rightNode != nil {
				t.nodeProvider.Release(rightNode.GetId(), false) 
			}
		} else if rightNode != nil && !rightNode.IsUnderflow() {
			log.Printf("Rebalancing node %d with right %d", node.GetId(), rightNode.GetId())
			sepKeyIdx := nodeIdx
			if node.IsLeaf() {
				key, value := rightNode.GetValueAt(0)
				node.InsertValueAt(node.NumKeys(), key, value)
				rightNode.Remove(0)
				
				parent.SetKeyAt(sepKeyIdx, key)
				lastParentChanged = true 
				
				t.nodeProvider.Release(node.GetId(), true)
				t.nodeProvider.Release(rightNode.GetId(), true)
			} else {
				key, childNodeId := rightNode.GetNodeAt(0)
				childNode := t.nodeProvider.GetNode(childNodeId)
				sepKey := parent.GetKeyAt(sepKeyIdx)

				node.InsertNodeAt(node.NumKeys(), sepKey, childNodeId)
				childNode.SetParent(node.GetId())
				rightNode.Remove(0)
				
				parent.SetKeyAt(sepKeyIdx, key)
				lastParentChanged = true
				
				t.nodeProvider.Release(node.GetId(), true)
				t.nodeProvider.Release(rightNode.GetId(), true)
				t.nodeProvider.Release(childNode.GetId(), true)
			}
		
			if leftNode != nil {
				t.nodeProvider.Release(leftNode.GetId(), false)
			}
		} else if leftNode != nil && t.CanMerge(node, leftNode) {
			log.Printf("Merging node %v with left node %v", node, leftNode)
			sepKeyIdx := nodeIdx - 1
			if node.IsLeaf() { 
				node.MergeLeaf(leftNode)
				parent.Remove(sepKeyIdx)
				llId, _ := leftNode.GetSibblings()

				if llId != NoPage {
					llNode := t.nodeProvider.GetNode(llId)
					node.SetLeftSibbling(llId)
					llNode.SetRightSibbling(node.GetId()) 

					t.nodeProvider.Release(llId, true)
				}

				t.nodeProvider.Release(node.GetId(), true)
				t.nodeProvider.Release(leftNode.GetId(), true)

				if rightNode != nil {
					t.nodeProvider.Release(rightNode.GetId(), false)	
				}
			} else {
				sepKey := parent.GetKeyAt(sepKeyIdx)
				node.MergeInternal(leftNode, sepKey)
				
				parent.Remove(sepKeyIdx)
				lastParentChanged = true 
				
				for i := 0; i < leftNode.NumKeys() + 1; i++ {
					_, childNodeId := leftNode.GetNodeAt(i)
					childNode := t.nodeProvider.GetNode(childNodeId)
					childNode.SetParent(node.GetId())

					t.nodeProvider.Release(childNodeId, true)
				}

				t.nodeProvider.Release(node.GetId(), true)
				t.nodeProvider.Release(leftNode.GetId(), true)

				if rightNode != nil {
					t.nodeProvider.Release(rightNode.GetId(), false)
				}
			}
		} else if rightNode != nil && t.CanMerge(node, rightNode) {
			log.Printf("Merging node %v with right node %v", node, rightNode)
			sepKeyIdx := nodeIdx
			if node.IsLeaf() {
				node.MergeLeaf(rightNode)
				
				parent.Remove(sepKeyIdx)
				lastParentChanged = true 

				_, rrNodeId := rightNode.GetSibblings()

				if rrNodeId != NoPage {
					rrNode := t.nodeProvider.GetNode(rrNodeId)
					rrNode.SetLeftSibbling(node.GetId())
					t.nodeProvider.Release(rrNodeId, true)
				}

				t.nodeProvider.Release(node.GetId(), true)
				t.nodeProvider.Release(rightNode.GetId(), true)

				if leftNode != nil {
					t.nodeProvider.Release(leftNode.GetId(), false)
				}
			} else {
				sepKey := parent.GetKeyAt(sepKeyIdx)
				node.MergeInternal(rightNode, sepKey)
				
				parent.Remove(sepKeyIdx)
				lastParentChanged = true 

				for i := 0; i < rightNode.NumKeys() + 1; i++ {
					_, childNodeId := rightNode.GetNodeAt(i)
					childNode := t.nodeProvider.GetNode(childNodeId)
					childNode.SetParent(node.GetId())

					t.nodeProvider.Release(childNodeId, true)
				}

				t.nodeProvider.Release(node.GetId(), true)
				t.nodeProvider.Release(rightNode.GetId(), true)

				if leftNode != nil {
					t.nodeProvider.Release(leftNode.GetId(), false)
				}
			}
		} else {
			if leftNode != nil {
				t.nodeProvider.Release(leftNode.GetId(), false)
			}

			if rightNode != nil {
				t.nodeProvider.Release(rightNode.GetId(), false)
			}

			t.nodeProvider.Release(node.GetId(), lastParentChanged)
			lastParentChanged = false
		} 
		lastReleased = it
	}

	for i := lastReleased + 1; i < len(underflowList); i += 1 {
		node := underflowList[i]
		t.nodeProvider.Release(node.GetId(), lastParentChanged)
		lastParentChanged = false
	}
}

func (t PagedBTree) CanMerge(left BTNode, right BTNode) bool {
	return false
}

func getSibblings(node BTNode, parent BTNode) (int, NodeId, NodeId) {
	var leftNode NodeId
	var rightNode NodeId
	
	sepKeyIdx, _ := parent.Search(node.GetKeyAt(0))
	
	if sepKeyIdx > 0 {
		_, leftNode = parent.GetNodeAt(sepKeyIdx - 1)
	} 
	
	if sepKeyIdx + 1< parent.NumKeys() {
		_, rightNode = parent.GetNodeAt(sepKeyIdx + 1)
	}

	return sepKeyIdx, leftNode, rightNode 
}

func (t *PagedBTree) Find(key Key) (Value, error){
	return Value{}, fmt.Errorf("not implemented")
} 

func (t *PagedBTree) find(key Key) (int, []BTNode) {
	root := t.nodeProvider.GetNode(t.masterPage.GetRootNode())
	
	node := root 
	stack := []BTNode{node}
	
	for !node.IsLeaf() {
		idx, _ := node.Search(key)
		_, childId := node.GetNodeAt(idx)
		node = t.nodeProvider.GetNode(childId)
		stack = append(stack, node) 
	}

	idx, _ := node.Search(key)
	slices.Reverse(stack)
	return idx, stack
}

