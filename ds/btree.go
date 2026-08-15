package ds

import (
	"fmt"
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
	
	for it, node := range underflowList {
		if it == len(underflowList) - 1 || !node.IsUnderflow() {
			break 
		}

		parent := underflowList[it + 1] 
		
		nodeIdx, left, right := getSibblings(node, parent)

		if leftNode != nil && !leftNode.IsUnderflow() {
			if node.IsLeaf() {
				key, value := leftNode.GetValueAt(leftNode.NumKeys() - 1)
				node.InsertValueAt(0, key, value)
				leftNode.Remove(leftNode.NumKeys() - 1)

				t.nodeProvider.Release(node.GetId(), true)
				t.nodeProvider.Release(leftNode.GetId(), true)
			} else {
				key, childNodeId := leftNode.GetNodeAt(leftNode.NumKeys() - 1)
				leftNode.Remove(leftNode.NumKeys() - 1)
				childNode := t.nodeProvider.GetNode(childNodeId)
				
				node.InsertNodeAt(0, key, childNodeId)
				childNode.SetParent(node.GetId())
			}
		} else if rightNode != nil && !rightNode.IsUnderflow() {
		}
	}
}


func getSibblings(node BTNode, parent BTNode) (int, NodeId, NodeId) {
	var left NodeId
	var right NodeId
	
		sepKeyIdx, _ := parent.Search(node.GetKeyAt(0))
		
		var leftNode, rightNode BTNode
		if sepKeyIdx > 0 {
			leftNode = t.nodeProvider.GetNode(parent.GetChildAt(sepKeyIdx - 1))
		} 
		if sepKeyIdx < parent.NumKeys() {
			rightNode = t.nodeProvider.GetNode(parent.GetChildAt(sepKeyIdx + 1))
		}
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
		childId := node.GetNodeAt(idx)
		node = t.nodeProvider.GetNode(childId)
		stack = append(stack, node) 
	}

	idx, _ := node.Search(key)
	slices.Reverse(stack)
	return idx, stack
}

