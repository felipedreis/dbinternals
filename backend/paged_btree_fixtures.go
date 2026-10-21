package backend 


import (
	"testing"
	"fmt"
)
// ---------------------------------------------------------------------------
// fakeNode: a minimal, byte-agnostic, in-memory stand-in for a Page-backed
// BTNode. It exists only so PagedBTree's orchestration logic (checkOverflow,
// checkUnderflow, tryBorrow, tryMerge, Find) can be driven and asserted on
// without a real BTNode implementation. It is not meant to model Page's
// byte layout or capacity limits.
// ---------------------------------------------------------------------------

type fakeNode struct {
	id       NodeId
	leaf     bool
	keys     []Key
	values   []Value
	children []NodeId

	parent NodeId
	left   NodeId
	right  NodeId

	maxKeys int
	minKeys int
}

func (n *fakeNode) GetId() NodeId          { return n.id }
func (n *fakeNode) GetParent() NodeId      { return n.parent }
func (n *fakeNode) SetParent(id NodeId)    { n.parent = id }
func (n *fakeNode) GetSibblings() (NodeId, NodeId) { return n.left, n.right }
func (n *fakeNode) SetLeftSibbling(id NodeId)      { n.left = id }
func (n *fakeNode) SetRightSibbling(id NodeId)     { n.right = id }

func (n *fakeNode) Init(key Key, leftChild, rightChild NodeId) {
	n.leaf = false
	n.keys = []Key{key}
	n.children = []NodeId{leftChild, rightChild}
}

// Search returns, for a leaf, the exact index of key (found=true) or the
// insertion point that keeps keys sorted (found=false) — a plain lower
// bound. For an internal node it returns the index of the child whose
// range contains key: the count of keys <= key, i.e. an upper bound, so
// that GetNodeAt(idx) (direct indexing) gives the correct child to
// descend into, and so sibling recovery in getSibblings works for any key
// drawn from a child's own range.
func (n *fakeNode) Search(key Key) (int, bool) {
	l, r := 0, len(n.keys)

	if n.leaf {
		for l < r {
			mid := (l + r) / 2
			switch key.Compare(n.keys[mid]) {
			case 0:
				return mid, true
			case 1:
				l = mid + 1
			default:
				r = mid
			}
		}
		return l, false
	}

	for l < r {
		mid := (l + r) / 2
		if key.Compare(n.keys[mid]) >= 0 {
			l = mid + 1
		} else {
			r = mid
		}
	}
	return l, false
}

func (n *fakeNode) InsertValueAt(idx int, key Key, value Value) {
	n.keys = InsertAt(n.keys, key, idx)
	n.values = InsertAt(n.values, value, idx)
}

// InsertNodeAt mirrors the current (unmodified) BTNode contract: the new
// child always lands at children[idx+1], i.e. to the right of the new
// key. That's correct for checkOverflow's split-insert and for
// right-borrow's append, but — as discussed — not for left-borrow's
// prepend, which needs the new child at children[idx] instead.
func (n *fakeNode) InsertNodeAt(idx int, key Key, child NodeId) {
	n.keys = InsertAt(n.keys, key, idx)
	n.children = InsertAt(n.children, child, idx+1)
}

func (n *fakeNode) AppendNode(key Key, child NodeId) {
	n.keys = InsertAt(n.keys, key, len(n.keys))
	n.children = InsertAt(n.children, child, len(n.children))
}


func (n *fakeNode) PrependNode(key Key, child NodeId) {
	n.keys = InsertAt(n.keys, key, 0)
	n.children = InsertAt(n.children, child, 0)
}

func (n *fakeNode) GetFirstNode() (Key, NodeId) {
	return n.keys[0], n.children[0]
}

func (n *fakeNode) GetLastNode() (Key, NodeId) {
	return n.keys[len(n.keys) - 1], n.children[len(n.children) - 1]
}

func (n *fakeNode) RemoveFirstNode() (Key, NodeId) {
	var k Key
	var c NodeId
	n.keys, k = Remove(n.keys, 0)
	n.children, c = Remove(n.children, 0)
	return k, c
}

func (n *fakeNode) RemoveLastNode() (Key, NodeId) {
	var k Key
	var c NodeId
	n.keys, k = Remove(n.keys, len(n.keys) - 1)
	n.children, c = Remove(n.children, len(n.children) - 1)

	return k, c
}

// RemoveNode drops keys[idx], plus whichever neighboring child is on side
// s: children[idx] when the removed separator's left neighbor is the one
// going away, children[idx+1] when it's the right neighbor. This is the
// direction-aware counterpart to InsertNodeAt, needed because merge()
// always has to drop the donor's (now-tombstoned) child specifically, and
// the donor can sit on either side of the separator.
func (n *fakeNode) RemoveNode(idx int, s side) (Key, NodeId) {
	var k Key
	var c NodeId
	n.keys, k = Remove(n.keys, idx)
	if s == left {
		n.children, c = Remove(n.children, idx)
	} else {
		n.children, c = Remove(n.children, idx+1)
	}
	return k, c
}

func (n *fakeNode) GetValueAt(idx int) (Key, Value) {
	return n.keys[idx], n.values[idx]
}

// GetNodeAt returns children[idx] directly, paired with keys[idx] when
// that exists (idx == NumKeys() addresses the rightmost child, which has
// no key to its right). This is the convention find()'s descent and
// getSibblings rely on.
func (n *fakeNode) GetNodeAt(idx int) (Key, NodeId) {
	var k Key
	if idx < len(n.keys) {
		k = n.keys[idx]
	}
	return k, n.children[idx]
}

func (n *fakeNode) IsAt(idx int, key Key) bool {
	if idx < 0 || idx >= len(n.keys) {
		return false
	}
	return n.keys[idx].Compare(key) == 0
}

func (n *fakeNode) SetKeyAt(idx int, key Key) { n.keys[idx] = key }
func (n *fakeNode) GetKeyAt(idx int) Key      { return n.keys[idx] }

// RemoveValue drops keys[idx] and values[idx]. Only ever called on leaves —
// an internal node's keys pair with children, not values, so removing one
// of its keys goes through RemoveNode instead.
func (n *fakeNode) RemoveValue(idx int) (Key, Value) {
	var v Value
	var k Key
	n.keys, k = Remove(n.keys, idx)
	n.values, v = Remove(n.values, idx)
	return k, v
}

func (n *fakeNode) IsLeaf() bool { return n.leaf }
func (n *fakeNode) NumKeys() int { return len(n.keys) }

func (n *fakeNode) IsOverflow() bool  { return len(n.keys) > n.maxKeys }
func (n *fakeNode) IsUnderflow() bool { return len(n.keys) < n.minKeys }

func (n *fakeNode) CanMergeLeaf(other BTNode) bool {
	return n.NumKeys()+other.NumKeys() <= n.maxKeys
}

func (n *fakeNode) MergeLeaf(other BTNode, s side) {
	o := other.(*fakeNode)
	if s == left {
		n.keys = append(append([]Key{}, o.keys...), n.keys...)
		n.values = append(append([]Value{}, o.values...), n.values...)
		return
	}
	n.keys = append(n.keys, o.keys...)
	n.values = append(n.values, o.values...)
}

func (n *fakeNode) CanMergeInternal(other BTNode, _ Key) bool {
	return n.NumKeys()+other.NumKeys()+1 <= n.maxKeys
}

func (n *fakeNode) MergeInternal(other BTNode, sepKey Key, s side) {
	o := other.(*fakeNode)
	if s == left {
		n.keys = append(append(append([]Key{}, o.keys...), sepKey), n.keys...)
		n.children = append(append([]NodeId{}, o.children...), n.children...)
		return
	}
	n.keys = append(append(append([]Key{}, n.keys...), sepKey), o.keys...)
	n.children = append(append([]NodeId{}, n.children...), o.children...)
}

func (n *fakeNode) Split(right BTNode) Key {
	r := right.(*fakeNode)
	r.leaf = n.leaf
	mid := len(n.keys) / 2

	if n.leaf {
		splitKey := n.keys[mid]
		r.keys = append(r.keys, n.keys[mid:]...)
		r.values = append(r.values, n.values[mid:]...)
		n.keys = n.keys[:mid]
		n.values = n.values[:mid]
		return splitKey
	}

	splitKey := n.keys[mid]
	r.keys = append(r.keys, n.keys[mid+1:]...)
	r.children = append(r.children, n.children[mid+1:]...)
	n.keys = n.keys[:mid]
	n.children = n.children[:mid+1]
	return splitKey
}

// ---------------------------------------------------------------------------
// fakeNodeProvider: an in-memory BTNodeProvider that also tracks pins, so
// tests can assert nothing was leaked, double-released, or touched after
// being tombstoned.
// ---------------------------------------------------------------------------

type fakeNodeProvider struct {
	nodeSize int
	nodes    map[NodeId]*fakeNode
	pins     map[NodeId]int
	dead     map[NodeId]bool
	nextId   NodeId
	errors   []string
}

func newFakeNodeProvider(nodeSize int) *fakeNodeProvider {
	return &fakeNodeProvider{
		nodeSize: nodeSize,
		nodes:    make(map[NodeId]*fakeNode),
		pins:     make(map[NodeId]int),
		dead:     make(map[NodeId]bool),
		nextId:   1,
	}
}

func (p *fakeNodeProvider) minKeys() int { return (p.nodeSize + 1) / 2 }

func (p *fakeNodeProvider) newNode(leaf bool) *fakeNode {
	id := p.nextId
	p.nextId++
	n := &fakeNode{id: id, leaf: leaf, maxKeys: p.nodeSize, minKeys: p.minKeys()}
	p.nodes[id] = n
	return n
}

// GetNewNode hands back a freshly allocated node already pinned once, the
// same contract GetNode has — callers always pair it with a Release (or
// Tombstone) once they're done. newNode itself stays unpinned, since test
// fixtures also call it directly to hand-build trees without going through
// that pinned-allocation path.
func (p *fakeNodeProvider) GetNewNode() BTNode {
	n := p.newNode(true)
	p.pins[n.id] = 1
	return n
}

func (p *fakeNodeProvider) GetNode(id NodeId) BTNode {
	if id == NoPage {
		return nil
	}
	if p.dead[id] {
		p.errorf("GetNode called on tombstoned node %d", id)
		return nil
	}
	n, ok := p.nodes[id]
	if !ok {
		p.errorf("GetNode called on unknown node %d", id)
		return nil
	}
	p.pins[id]++
	return n
}

func (p *fakeNodeProvider) Release(id NodeId, dirty bool) {
	if p.dead[id] {
		p.errorf("Release called on tombstoned node %d", id)
		return
	}
	if p.pins[id] <= 0 {
		p.errorf("Release called on node %d with no outstanding pin (double release?)", id)
		return
	}
	p.pins[id]--
}

func (p *fakeNodeProvider) Tombstone(id NodeId) {
	if p.dead[id] {
		p.errorf("Tombstone called twice on node %d", id)
		return
	}
	if p.pins[id] <= 0 {
		p.errorf("Tombstone called on node %d with no outstanding pin", id)
	} else {
		p.pins[id]--
	}
	p.dead[id] = true
}

func (p *fakeNodeProvider) errorf(format string, args ...any) {
	p.errors = append(p.errors, fmt.Sprintf(format, args...))
}

// AssertClean fails the test if any provider misuse was recorded (double
// release, use-after-tombstone, ...) or any node still has an outstanding
// pin (leaked without Release/Tombstone).
func (p *fakeNodeProvider) AssertClean(t *testing.T) {
	t.Helper()
	for _, msg := range p.errors {
		t.Errorf("node provider misuse: %s", msg)
	}
	for id, count := range p.pins {
		if count != 0 {
			t.Errorf("node %d has %d outstanding pin(s) — leaked without Release/Tombstone", id, count)
		}
	}
}

type fakeMasterPage struct {
	root NodeId
}

func (m *fakeMasterPage) GetRootNode() NodeId   { return m.root }
func (m *fakeMasterPage) SetRootNode(id NodeId) { m.root = id }

