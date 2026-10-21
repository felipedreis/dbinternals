package backend

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
	it := 0
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
		} else {
			t.reassignParent(split, split.GetId())
		}

		if it == len(splitList) - 1 {
			newRoot := t.nodeProvider.GetNewNode()
			newRoot.Init(splitKey, node.GetId(), split.GetId())
			node.SetParent(newRoot.GetId())
			split.SetParent(newRoot.GetId())
			t.masterPage.SetRootNode(newRoot.GetId())

			t.nodeProvider.Release(newRoot.GetId(), true)
			t.nodeProvider.Release(split.GetId(), true)
			break
		}

		parent := splitList[it + 1]

		idx, _ := parent.Search(splitKey)
		parent.InsertNodeAt(idx, splitKey, split.GetId())
		split.SetParent(parent.GetId())

		t.nodeProvider.Release(node.GetId(), true)
		t.nodeProvider.Release(split.GetId(), true)
		node = parent
		it++
	}

	/// parent didn't split but still dirty
	dirty := true

	// release the remaining nodes
	for ; it < len(splitList); it++ {
		t.nodeProvider.Release(splitList[it].GetId(), dirty)
		dirty = false
	}
}

func (t *PagedBTree) Remove(key Key) (Value, error)  {
	idx, underflowList := t.find(key)
	defer t.checkUnderflow(underflowList)

	if !underflowList[0].IsAt(idx, key) {
		return Value{}, fmt.Errorf("Key %v not found", key)
	}

	_, value := underflowList[0].RemoveValue(idx)
	return value, nil
}

func (t *PagedBTree) checkUnderflow(underflowList []BTNode) {
	parentDirty := false
	var it int
	var node BTNode
	var lastReleased int

	for it, node = range underflowList {
		if !node.IsUnderflow() {
			t.nodeProvider.Release(node.GetId(), parentDirty)
			parentDirty = false
			lastReleased = it
			break
		}

		if it == len(underflowList) - 1 {
			t.shrinkRoot(node, parentDirty)
			lastReleased = it
			break
		}

		parent := underflowList[it + 1]

		nodeIdx, leftId, rightId := getSibblings(node, parent)
		var leftNode, rightNode BTNode

		if leftId != NoPage {
			leftNode = t.nodeProvider.GetNode(leftId)
		}
		if rightId != NoPage {
			rightNode = t.nodeProvider.GetNode(rightId)
		}

		changed := t.tryBorrow(parent, node, leftNode, rightNode, nodeIdx)
		if !changed {
			changed = t.tryMerge(parent, node, leftNode, rightNode, nodeIdx)
		}

		if changed {
			parentDirty = true
			continue
		}

		t.nodeProvider.Release(node.GetId(), parentDirty)
		t.release(leftNode, false)
		t.release(rightNode, false)
		parentDirty = false
	}

	for i := lastReleased + 1; i < len(underflowList); i++ {
		t.nodeProvider.Release(underflowList[i].GetId(), parentDirty)
		parentDirty = false
	}
}

// shrinkRoot handles the top of the ancestor chain: an internal root that
// lost its last key gets replaced by its sole remaining child; anything
// else (a leaf root, or a root that's merely under threshold) is just
// released as a normal stop, since the root is exempt from occupancy
// requirements.
func (t *PagedBTree) shrinkRoot(root BTNode, dirty bool) {
	if root.NumKeys() == 0 && !root.IsLeaf() {
		_, childId := root.GetNodeAt(0)
		t.masterPage.SetRootNode(childId)
		t.nodeProvider.Tombstone(root.GetId())
		return
	}
	t.nodeProvider.Release(root.GetId(), dirty)
}

// release is Release with a built-in nil guard, since absent siblings are
// routinely nil BTNode handles.
func (t *PagedBTree) release(node BTNode, dirty bool) {
	if node != nil {
		t.nodeProvider.Release(node.GetId(), dirty)
	}
}

// tryBorrow attempts to pull one key (and, for internal nodes, one child)
// from a non-underflowing sibling into node. It returns whether it
// succeeded; on success node, the donor sibling and the untouched sibling
// (if any) have all been released.
func (t *PagedBTree) tryBorrow(parent, node, leftNode, rightNode BTNode, nodeIdx int) bool {
	if leftNode != nil && !leftNode.IsUnderflow() {
		t.borrow(parent, node, leftNode, nodeIdx, left)
		t.release(rightNode, false)
		return true
	}
	if rightNode != nil && !rightNode.IsUnderflow() {
		t.borrow(parent, node, rightNode, nodeIdx, right)
		t.release(leftNode, false)
		return true
	}
	return false
}

func (t *PagedBTree) borrow(parent, node, donor BTNode, nodeIdx int, s side) {
	log.Printf("Rebalancing node %d with %s sibling %d", node.GetId(), s, donor.GetId())
	sepKeyIdx := sepKeyIdxFor(nodeIdx, s)

	if node.IsLeaf() {

		if s == left {
			key, value := donor.RemoveValue(donor.NumKeys() - 1)
			node.InsertValueAt(0, key, value)
			parent.SetKeyAt(sepKeyIdx, key)
		} else {
			key, value := donor.RemoveValue(0)
			node.InsertValueAt(node.NumKeys(), key, value)
			parent.SetKeyAt(sepKeyIdx, donor.GetKeyAt(0))
		}
	} else {
		sepKey := parent.GetKeyAt(sepKeyIdx)
		var childId NodeId

		if s == left {
			var key Key
			key, childId = donor.RemoveLastNode()
			parent.SetKeyAt(sepKeyIdx, key)
			node.PrependNode(sepKey, childId)
		} else {
			_, childId = donor.RemoveFirstNode()
			node.AppendNode(sepKey, childId)
			parent.SetKeyAt(sepKeyIdx, donor.GetKeyAt(0))
		}

		child := t.nodeProvider.GetNode(childId)
		child.SetParent(node.GetId())
		t.nodeProvider.Release(childId, true)
	}

	t.nodeProvider.Release(node.GetId(), true)
	t.nodeProvider.Release(donor.GetId(), true)
}

// tryMerge attempts to merge node into a sibling that can't lend a key but
// has room to absorb node entirely, trying the left sibling first. It
// returns whether it succeeded; on success node, the donor sibling
// (tombstoned) and the untouched sibling (if any) have all been released.
func (t *PagedBTree) tryMerge(parent, node, leftNode, rightNode BTNode, nodeIdx int) bool {
	if leftNode != nil && t.merge(parent, node, leftNode, nodeIdx, left) {
		t.release(rightNode, false)
		return true
	}
	if rightNode != nil && t.merge(parent, node, rightNode, nodeIdx, right) {
		t.release(leftNode, false)
		return true
	}
	return false
}

func (t *PagedBTree) merge(parent, node, donor BTNode, nodeIdx int, s side) bool {
	sepKeyIdx := sepKeyIdxFor(nodeIdx, s)
	sepKey := parent.GetKeyAt(sepKeyIdx)

	switch {
	case node.IsLeaf() && node.CanMergeLeaf(donor):
		log.Printf("Merging node %v with %s node %v", node, s, donor)
		node.MergeLeaf(donor, s)
		parent.RemoveNode(sepKeyIdx, s)
		t.relinkLeafSiblings(node, donor, s)
	case !node.IsLeaf() && node.CanMergeInternal(donor, sepKey):
		log.Printf("Merging node %v with %s node %v", node, s, donor)
		node.MergeInternal(donor, sepKey, s)
		parent.RemoveNode(sepKeyIdx, s)
		t.reassignParent(donor, node.GetId())
	default:
		return false
	}

	t.nodeProvider.Release(node.GetId(), true)
	t.nodeProvider.Tombstone(donor.GetId())
	return true
}

// relinkLeafSiblings fixes the leaf-chain pointer on the far side of a
// merge: absorbing the left sibling means node's new left-left neighbor
// must point forward at node; absorbing the right sibling means the
// right-right neighbor must point back at node.
func (t *PagedBTree) relinkLeafSiblings(node, donor BTNode, s side) {
	if s == left {
		farId, _ := donor.GetSibblings()
		if farId == NoPage {
			return
		}
		far := t.nodeProvider.GetNode(farId)
		node.SetLeftSibbling(farId)
		far.SetRightSibbling(node.GetId())
		t.nodeProvider.Release(farId, true)
		return
	}

	_, farId := donor.GetSibblings()
	if farId == NoPage {
		return
	}
	far := t.nodeProvider.GetNode(farId)
	far.SetLeftSibbling(node.GetId())
	t.nodeProvider.Release(farId, true)
}

func (t *PagedBTree) reassignParent(mergedNode BTNode, parentId NodeId) {
	for i := 0; i < mergedNode.NumKeys() + 1; i++ {
		_, childNodeId := mergedNode.GetNodeAt(i)
		childNode := t.nodeProvider.GetNode(childNodeId)
		childNode.SetParent(parentId)

		t.nodeProvider.Release(childNodeId, true)
	}
}

// sepKeyIdxFor returns the parent's separator-key index for the sibling on
// the given side of the child at nodeIdx.
func sepKeyIdxFor(nodeIdx int, s side) int {
	if s == left {
		return nodeIdx - 1
	}
	return nodeIdx
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
	idx, stack := t.find(key)
	defer t.releaseNodes(stack)


	if !stack[0].IsAt(idx, key)  {
		return Value{}, fmt.Errorf("Key %v Not Found", key)
	} else {
		_, value := stack[0].GetValueAt(idx)
		return value, nil
	}
}

func (t *PagedBTree) releaseNodes(nodes []BTNode) {
	for _, node := range nodes {
		t.nodeProvider.Release(node.GetId(), false)
	}
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
