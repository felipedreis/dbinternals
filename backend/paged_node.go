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

func (pn *PagedNode) InsertNodeAt(idx int, key Key, node NodeId){
}

func (pn *PagedNode) AppendNode(key Key, node NodeId){
}

func (pn *PagedNode) PrependNode(key Key, nodeId NodeId){
}

func (pn *PagedNode) GetNodeAt(int) (key Key, nodeId NodeId){
	return nil, 0
}

func (pn *PagedNode) GetFirstNode() (key Key, nodeId NodeId){
	return nil, 0
}

func (pn *PagedNode) GetLastNode() (key Key, nodeId NodeId){
	return nil, 0
}

func (pn *PagedNode) RemoveFirstNode() (key Key, nodeId NodeId){
	return nil, 0 
}

func (pn *PagedNode) RemoveLastNode() (key Key, nodeId NodeId){
	return nil, 0 
}

func (pn *PagedNode) RemoveNode(idx int, s side) (key Key, nodeId NodeId){
	return nil, 0
}

func (pn *PagedNode) GetValueAt(idx int) (key Key, value Value){
	return nil, Value{}
}

func (pn *PagedNode) IsAt(idx int, key Key) bool{
	return false
}

func (pn *PagedNode) SetKeyAt(idx int, key Key){
}

func (pn *PagedNode) RemoveValue(idx int) (key Key, value Value){
	return nil, Value{}
}

func (pn *PagedNode) AppendValue(key Key, value Value)

func (pn *PagedNode) IsLeaf() bool{
	return false
}

func (pn *PagedNode) NumKeys() int {
	return 0
}

func (pn *PagedNode) GetKeyAt(idx int) Key{
	return nil
}

func (pn *PagedNode) IsOverflow() bool{
	return false
}

func (pn *PagedNode) IsUnderflow() bool{
	return false 
}

func (pn *PagedNode) CanMergeLeaf(node BTNode) bool{
	return  false
}

func (pn *PagedNode) MergeLeaf(node BTNode, s side){
}

func (pn *PagedNode) CanMergeInternal(node BTNode, sepKey Key) bool{
	paged := asPagedNode(node)
	return pn.page.GetFreeSpace() < paged.page.GetDataSize() + sepKey.Size()
}

func (pn *PagedNode) MergeInternal(node BTNode, key Key, s side){
}

func (pn *PagedNode) Split(node BTNode) Key{
	numKeys := pn.NumKeys() 
	paged := asPagedNode(node)
	size := pn.page.GetDataSize()

	if pn.IsLeaf() {
		for i := numKeys - 1; i >= 0 && paged.page.GetDataSize() < size/2 ; i-- {
			key := pn.GetKeyAt(i)
			value := pn.page.GetValueAt(i)
			node.AppendValue(key, Value{v: value})
		}

		
	
	} else {

	}

	return nil
}

func asPagedNode(btNode BTNode) *PagedNode {
	return btNode.(*PagedNode)
}
