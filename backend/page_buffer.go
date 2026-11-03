package backend

import (
	"log"
)

type PageBuffer struct {
	nodes    map[NodeId]PagedNode
	pins     map[NodeId]int
	dead     map[NodeId]bool
	nextId   NodeId
	fileManager FileManager
}

func NewPageBuffer(fileManager FileManager) *PageBuffer {
	return &PageBuffer{
		nodes:    make(map[NodeId]PagedNode),
		pins:     make(map[NodeId]int),
		dead:     make(map[NodeId]bool),
		fileManager: fileManager,
		nextId:   1,
	}
}

func (p *PageBuffer) newNode(leaf bool) BTNode {
	id := p.nextId
	p.nextId++
	n := MakePagedNode(id, leaf)
	p.nodes[id] = n
	return n
}

// GetNewNode hands back a freshly allocated node already pinned once, the
// same contract GetNode has — callers always pair it with a Release (or
// Tombstone) once they're done. newNode itself stays unpinned, since test
// fixtures also call it directly to hand-build trees without going through
// that pinned-allocation path.
func (p *PageBuffer) GetNewNode() BTNode {
	n := p.newNode(true)
	p.pins[n.GetId()] = 1
	return n
}

func (p *PageBuffer) GetNode(id NodeId) BTNode {
	if id == NoPage {
		return nil
	}
	n, ok := p.nodes[id]
	if !ok {
		n = p.fileManager.Read(NodeId(id))
		if n.page.IsToombstoned() {
			log.Printf("GetNode called on tombstoned node %d", id)
			return nil
		}
		log.Printf("GetNode called on unknown node %d\n", id)
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


