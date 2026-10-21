package backend

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type PageId uint64

const PAGE_SIZE int = 4096

type PageFlag byte

const (
	PAGE_ROOT PageFlag = iota
	PAGE_TYPE
	PAGE_HAS_OVERFLOW
	PAGE_IS_OVERFLOW
)

var Endian = binary.BigEndian

const (
	SZ_FLAGS        = 1
	SZ_VERSION      = 1
	SZ_KEYS         = 2
	SZ_DATA         = 2
	SZ_FRAGMENTED   = 2
	SZ_FREEBLOCK    = 2
	SZ_PAGE_POINTER = 8
)

const (
	PAGE_HEADER_FLAGS       = 0
	PAGE_HEADER_VERSION     = PAGE_HEADER_FLAGS + SZ_FLAGS            // 1
	PAGE_HEADER_KEYS        = PAGE_HEADER_VERSION + SZ_VERSION        // 2
	PAGE_HEADER_DATA_SIZE   = PAGE_HEADER_KEYS + SZ_KEYS              // 4
	PAGE_HEADER_FRAGMENTED  = PAGE_HEADER_DATA_SIZE + SZ_DATA         // 6
	PAGE_HEADER_FREEBLOCK   = PAGE_HEADER_FRAGMENTED + SZ_FRAGMENTED  // 8
	PAGE_HEADER_PARENT      = PAGE_HEADER_FREEBLOCK + SZ_FREEBLOCK    // 10
	PAGE_HEADER_LEFT_SIB    = PAGE_HEADER_PARENT + SZ_PAGE_POINTER    // 18
	PAGE_HEADER_RIGHT_SIB   = PAGE_HEADER_LEFT_SIB + SZ_PAGE_POINTER  // 26
	PAGE_HEADER_LEFT_CHILD  = PAGE_HEADER_RIGHT_SIB + SZ_PAGE_POINTER // 34
	PAGE_HEADER_SIZE        = PAGE_HEADER_LEFT_CHILD + SZ_PAGE_POINTER // 42
)

const (
	CELL_POINTER_SIZE       = 2
	CELL_KEY_SIZE           = 2
	CELL_DATA_SIZE          = 2
	CELL_CHILD_POINTER_SIZE = 8
)

/*
 page is formed by:
 [ page header | cell pointers array -> ... <- cell payloads ]

 page header fields:
 	- flags [1b]
 	- version [1b]
 	- keys count [2b]
 	- total cell data size [2b]
 	- fragmented space size [2b]
 	- free block offset [2b]
 	- parent id [8b]
 	- left sibling id [8b]
 	- right sibling id [8b]
 	- leftmost child id [8b] (for internal pages)

 internal node cell schema:
 [ key size (2b) | child pointer (8b) | key ]
 leaf node cell schema:
 [ key size (2b) | data size (2b) | key | data ]
*/

type Page struct {
	Id   PageId
	data []byte
}

func MakePage(id PageId, leaf bool) Page {
	page := Page{
		Id:   id,
		data: make([]byte, PAGE_SIZE),
	}

	page.SetFreeBlock(uint16(PAGE_SIZE))
	page.SetLeaf(leaf)

	return page
}

func (p *Page) Data() []byte {
	return p.data
}

func (p *Page) SetData(data []byte) {
	if len(data) != PAGE_SIZE {
		panic(fmt.Sprintf("invalid page size %d, expected %d", len(data), PAGE_SIZE))
	}
	copy(p.data, data)
}

func (p *Page) IsLeaf() bool {
	return p.data[PAGE_HEADER_FLAGS]&(1<<PAGE_TYPE) != 0
}

func (p *Page) IsRoot() bool {
	return p.data[PAGE_HEADER_FLAGS]&(1<<PAGE_ROOT) != 0
}

func (p *Page) HasOverflow() bool {
	return p.data[PAGE_HEADER_FLAGS]&(1<<PAGE_HAS_OVERFLOW) != 0
}

func (p *Page) IsOverflow() bool {
	return p.data[PAGE_HEADER_FLAGS]&(1<<PAGE_IS_OVERFLOW) != 0
}

func (p *Page) SetRoot(root bool) {
	if root {
		p.data[PAGE_HEADER_FLAGS] |= (1 << PAGE_ROOT)
	} else {
		p.data[PAGE_HEADER_FLAGS] &= ^byte(1 << PAGE_ROOT)
	}
}

func (p *Page) SetLeaf(leaf bool) {
	if leaf {
		p.data[PAGE_HEADER_FLAGS] |= (1 << PAGE_TYPE)
	} else {
		p.data[PAGE_HEADER_FLAGS] &= ^byte(1 << PAGE_TYPE)
	}
}

func (p *Page) SetHasOverflow(hasOverflow bool) {
	if hasOverflow {
		p.data[PAGE_HEADER_FLAGS] |= (1 << PAGE_HAS_OVERFLOW)
	} else {
		p.data[PAGE_HEADER_FLAGS] &= ^byte(1 << PAGE_HAS_OVERFLOW)
	}
}

func (p *Page) SetIsOverflow(isOverflow bool) {
	if isOverflow {
		p.data[PAGE_HEADER_FLAGS] |= (1 << PAGE_IS_OVERFLOW)
	} else {
		p.data[PAGE_HEADER_FLAGS] &= ^byte(1 << PAGE_IS_OVERFLOW)
	}
}

func (p *Page) GetPageVersion() int {
	return int(p.data[PAGE_HEADER_VERSION])
}

func (p *Page) SetPageVersion(version int) {
	p.data[PAGE_HEADER_VERSION] = byte(version)
}

func (p *Page) GetParent() PageId {
	return p.getPointer(PAGE_HEADER_PARENT)
}

func (p *Page) SetParent(parent PageId) {
	p.SetPointer(PAGE_HEADER_PARENT, parent)
}

func (p *Page) GetLeftSibbling() PageId {
	if !p.IsLeaf() {
		panic("Cannot get sibling in a non-leaf page")
	}
	return p.getPointer(PAGE_HEADER_LEFT_SIB)
}

func (p *Page) SetLeftSibbling(id PageId) {
	if !p.IsLeaf() {
		panic("Cannot set sibling in a non-leaf page")
	}
	p.SetPointer(PAGE_HEADER_LEFT_SIB, id)
}

func (p *Page) GetRightSibbling() PageId {
	if !p.IsLeaf() {
		panic("Cannot get sibling in a non-leaf page")
	}
	return p.getPointer(PAGE_HEADER_RIGHT_SIB)
}

func (p *Page) SetRightSibbling(id PageId) {
	if !p.IsLeaf() {
		panic("Cannot set sibling in a non-leaf page")
	}
	p.SetPointer(PAGE_HEADER_RIGHT_SIB, id)
}

func (p *Page) GetLeftChild() PageId {
	if p.IsLeaf() {
		panic("Cannot get left child in a leaf page")
	}
	return p.getPointer(PAGE_HEADER_LEFT_CHILD)
}

func (p *Page) SetLeftChild(id PageId) {
	if p.IsLeaf() {
		panic("Cannot set left child in a leaf page")
	}
	p.SetPointer(PAGE_HEADER_LEFT_CHILD, id)
}

func (p *Page) GetKeys() int {
	return int(p.getHeaderValue(PAGE_HEADER_KEYS, SZ_KEYS))
}

func (p *Page) SetKeys(keys int) {
	Endian.PutUint16(p.data[PAGE_HEADER_KEYS:PAGE_HEADER_KEYS+SZ_KEYS], uint16(keys))
}

func (p *Page) GetDataSize() uint16 {
	return p.getHeaderValue(PAGE_HEADER_DATA_SIZE, SZ_DATA)
}

func (p *Page) SetDataSize(data uint16) {
	Endian.PutUint16(p.data[PAGE_HEADER_DATA_SIZE:PAGE_HEADER_DATA_SIZE+SZ_DATA], data)
}

func (p *Page) GetFragmented() uint16 {
	return p.getHeaderValue(PAGE_HEADER_FRAGMENTED, SZ_FRAGMENTED)
}

func (p *Page) SetFragmented(fragmented uint16) {
	Endian.PutUint16(p.data[PAGE_HEADER_FRAGMENTED:PAGE_HEADER_FRAGMENTED+SZ_FRAGMENTED], fragmented)
}

func (p *Page) GetFreeBlockOffset() uint16 {
	return p.getHeaderValue(PAGE_HEADER_FREEBLOCK, SZ_FREEBLOCK)
}

func (p *Page) SetFreeBlock(freeblock uint16) {
	Endian.PutUint16(p.data[PAGE_HEADER_FREEBLOCK:PAGE_HEADER_FREEBLOCK+SZ_FREEBLOCK], freeblock)
}

func (p *Page) GetFreeSpace() uint16 {
	return p.GetFreeBlockSize() + p.GetFragmented()
}

func (p *Page) GetFreeBlockSize() uint16 {
	keys := p.GetKeys()
	keyArraySize := uint16(keys * CELL_POINTER_SIZE)
	offset := p.GetFreeBlockOffset()
	headerEnd := uint16(PAGE_HEADER_SIZE) + keyArraySize
	if offset < headerEnd {
		return 0
	}
	return offset - headerEnd
}

func (p *Page) GetKeyAt(idx int) []byte {
	if idx < 0 || idx >= p.GetKeys() {
		return nil
	}

	cellPointer := p.getCellPointer(idx)
	keySize := int(Endian.Uint16(p.data[cellPointer : cellPointer+CELL_KEY_SIZE]))

	var keyBegin int
	if p.IsLeaf() {
		keyBegin = int(cellPointer) + CELL_KEY_SIZE + CELL_DATA_SIZE
	} else {
		keyBegin = int(cellPointer) + CELL_KEY_SIZE + CELL_CHILD_POINTER_SIZE
	}

	return p.data[keyBegin : keyBegin+keySize]
}

func (p *Page) GetValueAt(idx int) []byte {
	if !p.IsLeaf() || idx < 0 || idx >= p.GetKeys() {
		return nil
	}

	cellPointer := p.getCellPointer(idx)
	dataSizeBegin := cellPointer + CELL_KEY_SIZE
	dataSizeEnd := cellPointer + CELL_KEY_SIZE + CELL_DATA_SIZE

	keySize := int(Endian.Uint16(p.data[cellPointer : cellPointer+CELL_KEY_SIZE]))
	dataSize := int(Endian.Uint16(p.data[dataSizeBegin:dataSizeEnd]))

	valueBegin := int(dataSizeEnd) + keySize
	return p.data[valueBegin : valueBegin+dataSize]
}

func (p *Page) GetChildAt(idx int) PageId {
	if p.IsLeaf() {
		return 0
	}

	if idx == 0 {
		return p.getPointer(PAGE_HEADER_LEFT_CHILD)
	}

	if idx < 0 || idx > p.GetKeys() {
		return 0
	}

	cellPointer := p.getCellPointer(idx - 1)
	childPointerBegin := cellPointer + CELL_KEY_SIZE
	return PageId(Endian.Uint64(p.data[childPointerBegin : childPointerBegin+CELL_CHILD_POINTER_SIZE]))
}

func (p *Page) PutKeyValue(idx int, key []byte, value []byte) bool {
	if !p.IsLeaf() {
		return false
	}

	cell, cellSize := encodeValueCell(key, value)
	return p.writeToFreeBlock(idx, cell, cellSize)
}

func (p *Page) PutKeyChild(idx int, key []byte, pageId PageId) bool {
	if p.IsLeaf() {
		return false
	}

	cell, cellSize := encodePointerCell(key, pageId)
	return p.writeToFreeBlock(idx, cell, cellSize)
}

func (p *Page) RemoveValueAtIdx(idx int) ([]byte, []byte, error) {
	if !p.IsLeaf() {
		return nil, nil, fmt.Errorf("cannot remove value from non-leaf page")
	}
	if idx < 0 || idx >= p.GetKeys() {
		return nil, nil, fmt.Errorf("index out of bound: %d", idx)
	}

	cell := p.removeIdx(idx)
	if cell == nil {
		return nil, nil, fmt.Errorf("failed to remove cell at index %d", idx)
	}

	key, val := decodeValueCell(cell)
	return key, val, nil
}

func (p *Page) RemoveChildAtIdx(idx int) ([]byte, PageId, error) {
	if p.IsLeaf() {
		return nil, 0, fmt.Errorf("cannot remove child from leaf page")
	}
	if idx < 0 || idx >= p.GetKeys() {
		return nil, 0, fmt.Errorf("index out of bound: %d", idx)
	}

	cell := p.removeIdx(idx)
	if cell == nil {
		return nil, 0, fmt.Errorf("failed to remove cell at index %d", idx)
	}

	key, childId := decodePointerCell(cell)
	return key, childId, nil
}

func (p *Page) Search(key []byte) (int, bool) {
	l := 0
	r := p.GetKeys()

	for l < r {
		mid := (l + r) / 2
		cmp := bytes.Compare(key, p.GetKeyAt(mid))
		if cmp == 0 {
			return mid, true
		} else if cmp > 0 {
			l = mid + 1
		} else {
			r = mid
		}
	}
	return l, false
}

func (p *Page) Split(right *Page) ([]byte, error) {
	if p.IsLeaf() != right.IsLeaf() {
		return nil, fmt.Errorf("cannot split between different page types")
	}

	keysCount := p.GetKeys()
	mid := keysCount / 2

	if p.IsLeaf() {
		splitKey := append([]byte(nil), p.GetKeyAt(mid)...)
		for i := mid; i < keysCount; i++ {
			k := p.GetKeyAt(i)
			v := p.GetValueAt(i)
			right.PutKeyValue(right.GetKeys(), k, v)
		}
		for i := keysCount - 1; i >= mid; i-- {
			if _, _, err := p.RemoveValueAtIdx(i); err != nil {
				return nil, err
			}
		}

		right.SetRightSibbling(p.GetRightSibbling())
		right.SetLeftSibbling(p.Id)
		p.SetRightSibbling(right.Id)

		return splitKey, nil
	} else {
		splitKey := append([]byte(nil), p.GetKeyAt(mid)...)
		right.SetLeftChild(p.GetChildAt(mid + 1))

		for i := mid + 1; i < keysCount; i++ {
			k := p.GetKeyAt(i)
			c := p.GetChildAt(i + 1)
			right.PutKeyChild(right.GetKeys(), k, c)
		}

		for i := keysCount - 1; i >= mid; i-- {
			if _, _, err := p.RemoveChildAtIdx(i); err != nil {
				return nil, err
			}
		}

		return splitKey, nil
	}
}

func (p *Page) Merge(right *Page, sepKey []byte) error {
	if p.IsLeaf() != right.IsLeaf() {
		return fmt.Errorf("cannot merge pages of different types")
	}

	if p.IsLeaf() {
		rightKeys := right.GetKeys()
		for i := 0; i < rightKeys; i++ {
			k := right.GetKeyAt(i)
			v := right.GetValueAt(i)
			if !p.PutKeyValue(p.GetKeys(), k, v) {
				return fmt.Errorf("failed to merge leaf key %v: out of space", k)
			}
		}
		p.SetRightSibbling(right.GetRightSibbling())
	} else {
		if !p.PutKeyChild(p.GetKeys(), sepKey, right.GetLeftChild()) {
			return fmt.Errorf("failed to merge sepKey %v: out of space", sepKey)
		}
		rightKeys := right.GetKeys()
		for i := 0; i < rightKeys; i++ {
			k := right.GetKeyAt(i)
			c := right.GetChildAt(i + 1)
			if !p.PutKeyChild(p.GetKeys(), k, c) {
				return fmt.Errorf("failed to merge internal key %v: out of space", k)
			}
		}
	}
	return nil
}

func (p *Page) writeToFreeBlock(idx int, cell []byte, cellSize uint16) bool {
	payloadSize := cellSize + CELL_POINTER_SIZE

	if payloadSize > p.GetFreeSpace() {
		return false
	}

	if payloadSize > p.GetFreeBlockSize() {
		p.defrag()
	}

	freeBlockPointer := p.GetFreeBlockOffset()
	cellOffset := freeBlockPointer - cellSize
	keys := p.GetKeys()
	keysSize := keys * CELL_POINTER_SIZE
	keyOffset := PAGE_HEADER_SIZE + idx*CELL_POINTER_SIZE

	copy(p.data[keyOffset+CELL_POINTER_SIZE:PAGE_HEADER_SIZE+keysSize+CELL_POINTER_SIZE], p.data[keyOffset:PAGE_HEADER_SIZE+keysSize])
	Endian.PutUint16(p.data[keyOffset:keyOffset+CELL_POINTER_SIZE], cellOffset)
	copy(p.data[cellOffset:cellOffset+cellSize], cell)

	keys++
	newDataSize := p.GetDataSize() + cellSize

	p.SetDataSize(newDataSize)
	p.SetKeys(keys)
	p.SetFreeBlock(cellOffset)
	return true
}

func (p *Page) defrag() {
	buffer := make([]byte, PAGE_SIZE)

	copy(buffer[0:PAGE_HEADER_SIZE], p.data[0:PAGE_HEADER_SIZE])
	keysSize := p.GetKeys()
	isLeaf := p.IsLeaf()

	freeBlockOffset := uint16(PAGE_SIZE)

	for i := 0; i < keysSize; i++ {
		cellPointer := p.getCellPointer(i)
		var cellSize uint16
		if isLeaf {
			keySize := Endian.Uint16(p.data[cellPointer : cellPointer+CELL_KEY_SIZE])
			dataSize := Endian.Uint16(p.data[cellPointer+CELL_KEY_SIZE : cellPointer+CELL_KEY_SIZE+CELL_DATA_SIZE])
			cellSize = CELL_KEY_SIZE + CELL_DATA_SIZE + keySize + dataSize
		} else {
			keySize := Endian.Uint16(p.data[cellPointer : cellPointer+CELL_KEY_SIZE])
			cellSize = CELL_KEY_SIZE + CELL_CHILD_POINTER_SIZE + keySize
		}

		freeBlockOffset -= cellSize
		copy(buffer[freeBlockOffset:freeBlockOffset+cellSize], p.data[cellPointer:cellPointer+cellSize])

		ptrOffset := PAGE_HEADER_SIZE + (i * CELL_POINTER_SIZE)
		Endian.PutUint16(buffer[ptrOffset:ptrOffset+CELL_POINTER_SIZE], freeBlockOffset)
	}

	Endian.PutUint16(buffer[PAGE_HEADER_FREEBLOCK:PAGE_HEADER_FREEBLOCK+SZ_FREEBLOCK], freeBlockOffset)
	Endian.PutUint16(buffer[PAGE_HEADER_FRAGMENTED:PAGE_HEADER_FRAGMENTED+SZ_FRAGMENTED], 0)

	copy(p.data, buffer)
}

func (p *Page) removeIdx(idx int) []byte {
	keys := p.GetKeys()
	if idx < 0 || idx >= keys {
		return nil
	}

	cellPointer := p.getCellPointer(idx)
	var cellSize uint16
	if p.IsLeaf() {
		keySize := Endian.Uint16(p.data[cellPointer : cellPointer+CELL_KEY_SIZE])
		dataSize := Endian.Uint16(p.data[cellPointer+CELL_KEY_SIZE : cellPointer+CELL_KEY_SIZE+CELL_DATA_SIZE])
		cellSize = CELL_KEY_SIZE + CELL_DATA_SIZE + keySize + dataSize
	} else {
		keySize := Endian.Uint16(p.data[cellPointer : cellPointer+CELL_KEY_SIZE])
		cellSize = CELL_KEY_SIZE + CELL_CHILD_POINTER_SIZE + keySize
	}

	cell := make([]byte, cellSize)
	copy(cell, p.data[cellPointer:cellPointer+cellSize])

	idxBegin := PAGE_HEADER_SIZE + CELL_POINTER_SIZE*idx
	nextIdxBegin := PAGE_HEADER_SIZE + CELL_POINTER_SIZE*(idx+1)
	keysEnd := PAGE_HEADER_SIZE + CELL_POINTER_SIZE*keys
	copy(p.data[idxBegin:keysEnd-CELL_POINTER_SIZE], p.data[nextIdxBegin:keysEnd])

	p.SetKeys(keys - 1)
	p.SetDataSize(p.GetDataSize() - cellSize)
	p.SetFragmented(p.GetFragmented() + cellSize)
	return cell
}

func encodeValueCell(key []byte, data []byte) ([]byte, uint16) {
	keySize := uint16(len(key))
	dataSize := uint16(len(data))
	cellSize := uint16(CELL_KEY_SIZE+CELL_DATA_SIZE) + keySize + dataSize

	buffer := make([]byte, cellSize)
	Endian.PutUint16(buffer[0:CELL_KEY_SIZE], keySize)
	Endian.PutUint16(buffer[CELL_KEY_SIZE:CELL_KEY_SIZE+CELL_DATA_SIZE], dataSize)

	keyOffset := CELL_KEY_SIZE + CELL_DATA_SIZE
	copy(buffer[keyOffset:keyOffset+int(keySize)], key)

	dataOffset := keyOffset + int(keySize)
	copy(buffer[dataOffset:dataOffset+int(dataSize)], data)

	return buffer, cellSize
}

func decodeValueCell(cell []byte) ([]byte, []byte) {
	keySize := int(Endian.Uint16(cell[0:CELL_KEY_SIZE]))
	dataSize := int(Endian.Uint16(cell[CELL_KEY_SIZE : CELL_KEY_SIZE+CELL_DATA_SIZE]))

	keyOffset := CELL_KEY_SIZE + CELL_DATA_SIZE
	key := cell[keyOffset : keyOffset+keySize]

	dataOffset := keyOffset + keySize
	data := cell[dataOffset : dataOffset+dataSize]

	return key, data
}

func encodePointerCell(key []byte, pointer PageId) ([]byte, uint16) {
	keySize := uint16(len(key))
	cellSize := uint16(CELL_KEY_SIZE+CELL_CHILD_POINTER_SIZE) + keySize

	buffer := make([]byte, cellSize)
	Endian.PutUint16(buffer[0:CELL_KEY_SIZE], keySize)
	Endian.PutUint64(buffer[CELL_KEY_SIZE:CELL_KEY_SIZE+CELL_CHILD_POINTER_SIZE], uint64(pointer))

	keyOffset := CELL_KEY_SIZE + CELL_CHILD_POINTER_SIZE
	copy(buffer[keyOffset:keyOffset+int(keySize)], key)

	return buffer, cellSize
}

func decodePointerCell(cell []byte) ([]byte, PageId) {
	keySize := int(Endian.Uint16(cell[0:CELL_KEY_SIZE]))
	pointer := PageId(Endian.Uint64(cell[CELL_KEY_SIZE : CELL_KEY_SIZE+CELL_CHILD_POINTER_SIZE]))

	keyOffset := CELL_KEY_SIZE + CELL_CHILD_POINTER_SIZE
	key := cell[keyOffset : keyOffset+keySize]

	return key, pointer
}

func (p *Page) getCellPointer(idx int) uint16 {
	begin := PAGE_HEADER_SIZE + CELL_POINTER_SIZE*idx
	end := begin + CELL_POINTER_SIZE
	return Endian.Uint16(p.data[begin:end])
}

func (p *Page) getPointer(offset int) PageId {
	return PageId(Endian.Uint64(p.data[offset : offset+SZ_PAGE_POINTER]))
}

func (p *Page) SetPointer(pointerOffset int, val PageId) {
	Endian.PutUint64(p.data[pointerOffset:pointerOffset+SZ_PAGE_POINTER], uint64(val))
}

func (p *Page) getHeaderValue(begin int, offset int) uint16 {
	return Endian.Uint16(p.data[begin : begin+offset])
}
