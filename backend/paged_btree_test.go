package backend

import (
	"fmt"
	"math/rand/v2"
	"testing"
)

func newFakePagedBTree(nodeSize int) (*PagedBTree, *fakeNodeProvider, *fakeMasterPage) {
	provider := newFakeNodeProvider(nodeSize)
	root := provider.newNode(true)
	master := &fakeMasterPage{root: root.id}
	tree := &PagedBTree{masterPage: master, nodeProvider: provider}
	return tree, provider, master
}

// ---------------------------------------------------------------------------
// structural validation, mirroring validateTree's role for InMemoryBTree
// ---------------------------------------------------------------------------

func validatePagedTree(t *testing.T, provider *fakeNodeProvider, rootId NodeId) {
	t.Helper()
	validatePagedNode(t, provider, rootId, map[NodeId]bool{})
}

func validatePagedNode(t *testing.T, provider *fakeNodeProvider, id NodeId, seen map[NodeId]bool) {
	t.Helper()
	if seen[id] {
		t.Fatalf("node %d visited twice while validating tree — cycle or shared child", id)
	}
	seen[id] = true

	n, ok := provider.nodes[id]
	if !ok || provider.dead[id] {
		t.Fatalf("node %d is referenced but missing or tombstoned", id)
	}

	for i := 1; i < len(n.keys); i++ {
		if n.keys[i-1].Compare(n.keys[i]) >= 0 {
			t.Fatalf("node %d has keys out of order at %d,%d: %v", id, i-1, i, n.keys)
		}
	}

	if n.leaf {
		if len(n.children) != 0 {
			t.Fatalf("leaf node %d unexpectedly has children", id)
		}
		return
	}

	if len(n.children) != len(n.keys)+1 {
		t.Fatalf("internal node %d has %d keys and %d children", id, len(n.keys), len(n.children))
	}

	for _, childId := range n.children {
		child, ok := provider.nodes[childId]
		if !ok || provider.dead[childId] {
			t.Fatalf("node %d references missing or tombstoned child %d", id, childId)
		}
		if child.parent != id {
			t.Fatalf("node %d's child %d has parent %d, want %d", id, childId, child.parent, id)
		}
		validatePagedNode(t, provider, childId, seen)
	}
}

// ---------------------------------------------------------------------------
// Add / split coverage
// ---------------------------------------------------------------------------

func TestPagedBTree_Add(t *testing.T) {
	tree, provider, master := newFakePagedBTree(3)

	tests := []struct {
		name string
		key  int
	}{
		{"First insert, initialize root node", 10},
		{"Second insert", 8},
		{"Third insert", 5},
		{"Splits root node", 1},
		{"Insert into the left leaf node", 7},
		{"Insert into the left leaf node, splits", 2},
		{"Inserts into the rightmost node", 20},
		{"Inserts into the rightmost node, splits", 30},
		{"Saturates the last leaf", 25},
		{"Propagates split to the root", 35},
		{"Inserts to the leftmost node", 3},
		{"Inserts to the leftmost node, splits", 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := TestKey{k: tt.key}
			value := Value{v: []byte(fmt.Sprintf("val%d", tt.key))}
			tree.Add(key, value)

			validatePagedTree(t, provider, master.root)

			got, err := tree.Find(key)
			if err != nil {
				t.Fatalf("Find(%v) returned error: %v", key, err)
			}
			if string(got.v) != string(value.v) {
				t.Fatalf("Find(%v) = %q, want %q", key, got.v, value.v)
			}
		})
	}

	provider.AssertClean(t)
}

func TestPagedBTree_InsertRandomOrder_PreserveProperties(t *testing.T) {
	const nodeSize = 4
	const testCases = 5 * nodeSize

	tree, provider, master := newFakePagedBTree(nodeSize)

	for _, i := range rand.Perm(testCases) {
		key := TestKey{k: i}
		value := Value{v: []byte(fmt.Sprintf("val%d", i))}
		t.Run(fmt.Sprintf("Adding key %d", i), func(t *testing.T) {
			tree.Add(key, value)
			validatePagedTree(t, provider, master.root)
		})
	}

	provider.AssertClean(t)
}

// ---------------------------------------------------------------------------
// Remove coverage: scenarios kept to the side that's known to work (borrow
// from / merge with the right sibling). Left-side internal rebalancing is
// exercised separately below, where it's expected to fail.
// ---------------------------------------------------------------------------

func TestPagedBTree_Remove_SingleLeafRoot(t *testing.T) {
	tree, provider, master := newFakePagedBTree(4)

	keys := []int{10, 5, 20, 15}
	for _, k := range keys {
		tree.Add(TestKey{k: k}, Value{v: []byte(fmt.Sprintf("val%d", k))})
	}
	validatePagedTree(t, provider, master.root)

	for _, k := range keys {
		testKey := TestKey{k: k}
		if _, err := tree.Remove(testKey); err != nil {
			t.Fatalf("Remove(%v) returned error: %v", testKey, err)
		}
		if _, err := tree.Find(testKey); err == nil {
			t.Fatalf("Find(%v) unexpectedly succeeded after removal", testKey)
		}
	}

	provider.AssertClean(t)
}

// TestPagedBTree_Remove_BorrowAndMergeFromRight builds a three-leaf tree
// by hand and removes keys from the leftmost leaf, which only ever needs
// to borrow from or merge with its right sibling.
//
// The leftmost leaf is used deliberately: with three children, the
// *middle* child (the one right before the last) is the one that trips
// getSibblings' off-by-one (see below) when looking for its own right
// sibling — `sepKeyIdx+1 < parent.NumKeys()` excludes the case where the
// right sibling is the parent's very last child, i.e. sepKeyIdx+1 ==
// NumKeys(). The leftmost child's sepKeyIdx is 0, which never hits that
// boundary here, so this test stays clear of that separate bug.
func TestPagedBTree_Remove_BorrowAndMergeFromRight(t *testing.T) {
	tree, provider, master := newFakePagedBTree(4)

	leaf0 := provider.newNode(true)
	leaf0.keys = []Key{TestKey{1}, TestKey{2}}
	leaf0.values = []Value{{v: []byte("v1")}, {v: []byte("v2")}}

	leaf1 := provider.newNode(true)
	leaf1.keys = []Key{TestKey{10}, TestKey{11}}
	leaf1.values = []Value{{v: []byte("v10")}, {v: []byte("v11")}}

	leaf2 := provider.newNode(true)
	leaf2.keys = []Key{TestKey{20}, TestKey{21}, TestKey{22}}
	leaf2.values = []Value{{v: []byte("v20")}, {v: []byte("v21")}, {v: []byte("v22")}}

	leaf0.right, leaf1.left = leaf1.id, leaf0.id
	leaf1.right, leaf2.left = leaf2.id, leaf1.id

	root := provider.newNode(false)
	root.keys = []Key{TestKey{10}, TestKey{20}}
	root.children = []NodeId{leaf0.id, leaf1.id, leaf2.id}
	leaf0.parent, leaf1.parent, leaf2.parent = root.id, root.id, root.id

	master.root = root.id

	validatePagedTree(t, provider, master.root)

	for _, k := range []int{1, 2} {
		testKey := TestKey{k: k}
		if _, err := tree.Remove(testKey); err != nil {
			t.Fatalf("Remove(%v) returned error: %v", testKey, err)
		}
		validatePagedTree(t, provider, master.root)
	}

	for _, k := range []int{10, 11, 20, 21, 22} {
		if _, err := tree.Find(TestKey{k: k}); err != nil {
			t.Fatalf("Find(%v) failed after borrow/merge: %v", TestKey{k: k}, err)
		}
	}

	provider.AssertClean(t)
}

// ---------------------------------------------------------------------------
// The bug: tryBorrow's internal, borrow-from-left case asks InsertNodeAt
// to put the donated child to the LEFT of the new key (children[0]), but
// InsertNodeAt always places it to the right (children[idx+1]) — the same
// convention checkOverflow and right-borrow correctly rely on. This test
// builds exactly that scenario directly (bypassing Add/Remove) and checks
// the resulting child order on `node`.
// ---------------------------------------------------------------------------

func TestPagedBTree_Borrow_InternalFromLeft_ChildOrder(t *testing.T) {
	tree, provider, _ := newFakePagedBTree(4)

	d0 := provider.newNode(true)
	d0.keys = []Key{TestKey{1}}
	d1 := provider.newNode(true)
	d1.keys = []Key{TestKey{15}}
	d2 := provider.newNode(true)
	d2.keys = []Key{TestKey{25}} // donor's last child — should become node's new first child

	donor := provider.newNode(false)
	donor.keys = []Key{TestKey{10}, TestKey{20}}
	donor.children = []NodeId{d0.id, d1.id, d2.id}

	n0 := provider.newNode(true)
	n0.keys = []Key{TestKey{60}}
	n1 := provider.newNode(true)
	n1.keys = []Key{TestKey{80}}

	node := provider.newNode(false)
	node.keys = []Key{TestKey{70}}
	node.children = []NodeId{n0.id, n1.id}

	parent := provider.newNode(false)
	parent.keys = []Key{TestKey{50}}
	parent.children = []NodeId{donor.id, node.id}

	// node is parent's child at index 1 (donor is at index 0).
	tree.borrow(parent, node, donor, 1, left)

	wantFirstChild := d2.id
	_, gotFirstChild := node.GetNodeAt(0)
	if gotFirstChild != wantFirstChild {
		t.Errorf("node's new first child = %d, want %d (donor's last child, borrowed from the left)",
			gotFirstChild, wantFirstChild)
	}

	wantSecondChild := n0.id
	_, gotSecondChild := node.GetNodeAt(1)
	if gotSecondChild != wantSecondChild {
		t.Errorf("node's second child = %d, want %d (node's own original first child, shifted right)",
			gotSecondChild, wantSecondChild)
	}

	if got := node.GetKeyAt(0); got.Compare(TestKey{50}) != 0 {
		t.Errorf("node's new first key = %v, want the old parent separator %v", got, TestKey{50})
	}
}
