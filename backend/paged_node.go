package backend

type PagedNode struct {
	id NodeId
	page Page
}


func MakePagedNode(id NodeId, leaf bool) PagedNode {
	return PagedNode{id: NodeId(id), page: MakePage(PageId(id), leaf)}
}

func (pn *PagedNode) Init(key Key, left NodeId, right NodeId)  {
	pn.page.SetLeftChild(PageId(left))
	pn.page.PutKeyChild(0, key.Bytes(), PageId(right)) 
}

func (pn *PagedNode) GetId() NodeId {
	return pn.id
}

func (pn *PagedNode) GetParent() NodeId {
	return NodeId(pn.page.GetParent())
}

func (pn *PagedNode) SetParent(nodeId NodeId){
	pn.page.SetParent(PageId(nodeId))
}

func (pn *PagedNode) GetSibblings() (NodeId, NodeId){
	return NodeId(pn.page.GetLeftSibbling()), NodeId(pn.page.GetRightSibbling())
}

func (pn *PagedNode) SetLeftSibbling(nodeId NodeId) {
	pn.page.SetLeftSibbling(PageId(nodeId))
}
func (pn *PagedNode) SetRightSibbling(nodeId NodeId) {
	pn.page.SetRightSibbling(PageId(nodeId))
}
func (pn *PagedNode) Search(key Key) (int, bool){
	return pn.page.Search(key.Bytes())
}
func (pn *PagedNode) InsertValueAt(idx int, key Key, value Value){
	pn.page.PutKeyValue(idx, key.Bytes(), value.v)
}
func (pn *PagedNode) InsertNodeAt(int, Key, NodeId){
}
func (pn *PagedNode) AppendNode(Key, NodeId){
}
func (pn *PagedNode) PrependNode(Key, NodeId){
}
func (pn *PagedNode) GetNodeAt(int) (Key, NodeId){
	return nil, 0
}

func (pn *PagedNode) GetFirstNode() (Key, NodeId){
	return nil, 0
}

func (pn *PagedNode) GetLastNode() (Key, NodeId){
	return nil, 0
}

func (pn *PagedNode) RemoveFirstNode() (Key, NodeId){
	return nil, 0 
}

func (pn *PagedNode) RemoveLastNode() (Key, NodeId){
	return nil, 0 
}

func (pn *PagedNode) RemoveNode(int, side) (Key, NodeId){
	return nil, 0
}

func (pn *PagedNode) GetValueAt(int) (Key, Value){
	return nil, Value{}
}

func (pn *PagedNode) IsAt(int,Key) bool{
	return false
}

func (pn *PagedNode) SetKeyAt(int, Key){
}

func (pn *PagedNode) RemoveValue(int) (Key, Value){
	return nil, Value{}
}

func (pn *PagedNode) IsLeaf() bool{
	return false
}

func (pn *PagedNode) NumKeys() int {
	return 0
}

func (pn *PagedNode) GetKeyAt(int) Key{
	return nil
}

func (pn *PagedNode) IsOverflow() bool{
	return false
}

func (pn *PagedNode) IsUnderflow() bool{
	return false 
}

func (pn *PagedNode) CanMergeLeaf(BTNode) bool{
	return  false
}

func (pn *PagedNode) MergeLeaf(BTNode, side){
}

func (pn *PagedNode) CanMergeInternal(BTNode, Key) bool{
	return false
}

func (pn *PagedNode) MergeInternal(BTNode, Key, side){
}

func (pn *PagedNode) Split(BTNode) Key{
	return nil
}
