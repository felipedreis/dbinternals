package ds

import "fmt" 

type NodeId uint64

const NoPage NodeId = 0

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
	GetValueAt(int) (Key, Value)
	GetNodeAt(int) (Key, NodeId)
	IsAt(int,Key) bool
	Remove(int) Value
	IsLeaf() bool
	NumKeys() int 
	GetKeyAt(int) Key
	
	IsOverflow() bool
	IsUnderflow() bool
	Merge(BTNode)
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

