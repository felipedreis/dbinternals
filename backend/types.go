package backend

import "fmt" 

type NodeId uint64

const NoPage NodeId = 0

// side identifies which neighbor of a node an operation targets.
type side bool

const (
	left  side = true
	right side = false
)

func (s side) String() string {
	if s == left {
		return "left"
	}
	return "right"
}

type BTNode interface {
	GetId() NodeId 
	GetParent() NodeId 
	SetParent(NodeId)
	GetSibblings() (NodeId, NodeId)
	SetLeftSibbling(NodeId)
	SetRightSibbling(NodeId)
	Init(Key, NodeId, NodeId)  
	Search(Key) (int, bool)
	InsertValueAt(int, Key, Value)
	InsertNodeAt(int, Key, NodeId)
	AppendNode(Key, NodeId)
	PrependNode(Key, NodeId)
	GetNodeAt(int) (Key, NodeId)
	
	// Used in te borrow from left/right
	GetFirstNode() (Key, NodeId)
	GetLastNode() (Key, NodeId)
	RemoveFirstNode() (Key, NodeId)
	RemoveLastNode() (Key, NodeId)
	RemoveNode(int, side) (Key, NodeId)

	GetValueAt(int) (Key, Value)
	IsAt(int,Key) bool
	SetKeyAt(int, Key)
	RemoveValue(int) (Key, Value)
	IsLeaf() bool
	NumKeys() int 
	GetKeyAt(int) Key
	
	IsOverflow() bool
	IsUnderflow() bool
	CanMergeLeaf(BTNode) bool
	MergeLeaf(BTNode, side)
	CanMergeInternal(BTNode, Key) bool
	MergeInternal(BTNode, Key, side)
	Split(BTNode) Key
}


type BTNodeProvider interface {
	GetNewNode() BTNode
	GetNode(NodeId) BTNode 
	Release(NodeId, bool)
	Tombstone(NodeId) 
}

type Key interface {
	fmt.Stringer
	Compare(other Key) int
}

type Value struct {
	v []byte
}

type BTree interface {
	Add(Key, Value) 
	Remove(Key) (Value, error)  
	Find(Key) (Value, error)
}

