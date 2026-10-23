package backend



type PageBuffer struct {
	nodes    map[NodeId]*BTNode
	pins     map[NodeId]int
	dead     map[NodeId]bool
	nextId   NodeId
}

func NewPageBuffer(fileManager FileManager) *PageBuffer {
	return &PageBuffer{
		nodeSize: nodeSize,
		nodes:    make(map[NodeId]*fakeNode),
		pins:     make(map[NodeId]int),
		dead:     make(map[NodeId]bool),
		nextId:   1,
	}
}

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

