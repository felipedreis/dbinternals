package filesystem

import (
	"encoding/binary"
)

type PageId uint64

const PAGE_SIZE int = 4096

type PageFlag byte

const (
	PAGE_ROOT PageFlag = iota 
	PAGE_TYPE
	PAGE_HAS_OVERFLOW
	PAGE_IS_OVERLFOW
)

var Endian =  binary.BigEndian

const (
	SZ_FLAGS = 1
	SZ_VERSION = 1
	SZ_KEYS = 2
	SZ_DATA = 2
	SZ_FRAGMENTED = 2
	SZ_FREEBLOCK = 2
	SZ_PAGE_POINTER = 8
	
)

const (
	PAGE_HEADER_FLAGS 			  	= 0 
	PAGE_HEADER_VERSION           	= PAGE_HEADER_FLAGS + SZ_FLAGS
	PAGE_HEADER_KEYS		  		= PAGE_HEADER_VERSION + SZ_VERSION 
	PAGE_HEADER_DATA_SIZE		  	= PAGE_HEADER_KEYS + SZ_KEYS 
	PAGE_HEADER_FRAGMENTED 			= PAGE_HEADER_DATA_SIZE + SZ_DATA
	PAGE_HEADER_FREEBLOCK     		= PAGE_HEADER_FRAGMENTED + SZ_FRAGMENTED 
	PAGE_HEADER_PARENT 				= PAGE_HEADER_FREEBLOCK + SZ_FREEBLOCK
	PAGE_HEADER_LEFT_SIB 			= PAGE_HEADER_PARENT + SZ_PAGE_POINTER
	PAGE_HEADER_RIGHT_SIB     		= PAGE_HEADER_LEFT_SIB + SZ_PAGE_POINTER
	PAGE_HEADER_LEFT_CHILD  		= PAGE_HEADER_RIGHT_SIB + SZ_PAGE_POINTER
	PAGE_HEADER_SIZE 			  	= PAGE_HEADER_LEFT_CHILD + SZ_PAGE_POINTER
)

const (
	CELL_POINTER_SIZE 	    = 2
	CELL_KEY_SIZE           = 2
	CELL_DATA_SIZE          = 2
	CELL_CHILD_POINTER_SIZE = 8
)


/*
 page is formed by 
 [ page header | page cells  ]

 a page header indicates: 
 	- the page flags 										[2b]
	- the number of keys  									[2b]
	- the parent id 										[4b]
	- the left and right sibblings (in case of leaf node)	[8b]
	- the leftmost child 									[4b]

then we have to store key/pointers in case of internal nodes and 
key/values in case of leaf nodes.
if it's a internal node we'll store pointers as 
 [ key size (2b) | pointer (4b) | key ] 

 in a inernal node we have K keys and K+1 pointers. In our b+tree implementation 
 we have an invariant that for the Ki, all the keys under the right pointer are 
 strictly >= Ki. This leaves us with the left child, that we'll keep in the 
 final 4 bytes of the header  

 for leaf nodes we will store the data in the following schema 

 [ key size (2b) | data size (2b) | key | data ]
*/

type Page struct {
	Id PageId
	data []byte
}


func (p *Page) IsLeaf() bool {
	return p.data[PAGE_HEADER_FLAGS] & 1 << PAGE_TYPE != 0
}

func (p *Page) IsRoot() bool {
	return p.data[PAGE_HEADER_FLAGS] & 1 << PAGE_ROOT != 0
}

func (p *Page) HasOverflow() bool {
	return p.data[PAGE_HEADER_FLAGS] & 1 << PAGE_HAS_OVERFLOW != 0
}

func (p *Page) IsOverflow() bool {
	return p.data[PAGE_HEADER_FLAGS] & 1 <<  PAGE_IS_OVERLFOW != 0 
}

func (p *Page) GetPageVersion() int {
	return int(p.data[PAGE_HEADER_VERSION])
}

func (p *Page) GetParent() PageId {
	id := p.getHeaderValue(PAGE_HEADER_PARENT, SZ_PAGE_POINTER)
	return PageId(id) 
}


func (p *Page) GetLeftSibbling() PageId {
	if !p.IsLeaf() {
		panic("Cannot get sibbling in a non-leaf page")
	}
	
	return p.getPointer(PAGE_HEADER_LEFT_SIB)
}

func (p *Page) GetRightSibbling() PageId { 
	if !p.IsLeaf() {
		panic("Cannot get sibbling in a non-leaf page")
	}
	
	return p.getPointer(PAGE_HEADER_RIGHT_SIB)
}

func (p *Page) GetKeys() uint16 {
	keys := p.getHeaderValue(PAGE_HEADER_KEYS, SZ_KEYS)
	return keys
}

func (p *Page) GetDataSize() uint16 {
	return p.getHeaderValue(PAGE_HEADER_DATA_SIZE, SZ_DATA) 
}

func (p *Page) GetFreeSpace() uint16 {
	return p.GetFreeBlockSize() + p.getHeaderValue(PAGE_HEADER_FRAGMENTED, SZ_FRAGMENTED)
}

func (p *Page) GetFreeBlockSize() uint16 {
	keys := p.GetKeys() 
	keyArraySize := uint16(keys * CELL_KEY_SIZE)
	
	return p.GetFreeBlockOffset() - (PAGE_HEADER_SIZE + keyArraySize)
}

func (p *Page) GetFreeBlockOffset() uint16 {
	return p.getHeaderValue(PAGE_HEADER_FREEBLOCK, SZ_FREEBLOCK)
}

func (p *Page) GetKeyAt(idx int) []byte {
	
	var keyBegin, keySize int
	cellPointer := p.getCellPointer(idx)
	binary.Decode(p.data[cellPointer:cellPointer+2], Endian, &keySize)
	
	if p.IsLeaf() {
		keyBegin = int(cellPointer) + CELL_KEY_SIZE + CELL_DATA_SIZE
	} else {
		keyBegin = int(cellPointer) + CELL_KEY_SIZE + CELL_CHILD_POINTER_SIZE
	}
	
	return p.data[keyBegin:keyBegin+keySize]
}

func (p *Page) SetKeys(keys uint16) {
	Endian.PutUint16(p.data[PAGE_HEADER_KEYS:PAGE_HEADER_KEYS+SZ_KEYS], keys); 
}

func (p *Page) SetFreeBlock(freeblock uint16) {
	Endian.PutUint16(p.data[PAGE_HEADER_FREEBLOCK:PAGE_HEADER_FREEBLOCK+SZ_FREEBLOCK], freeblock)
}

func (p *Page) SetFragmented(fragmented uint16) {
	Endian.PutUint16(p.data[PAGE_HEADER_FRAGMENTED:PAGE_HEADER_FRAGMENTED+SZ_FRAGMENTED], fragmented)
}

func (p *Page) SetDataSize(data uint16) {
	Endian.PutUint16(p.data[PAGE_HEADER_DATA_SIZE:PAGE_HEADER_DATA_SIZE+SZ_DATA], data)
}

func (p *Page) SetPointer(pointerOffset int, val PageId) {
	Endian.PutUint64(p.data[pointerOffset:pointerOffset+SZ_PAGE_POINTER], uint64(val))
}

func (p *Page) GetValueAt(idx int) []byte {
	if !p.IsLeaf() {
		return nil
	}

	cellPointer := p.getCellPointer(idx) 
	dataSizeBegin := cellPointer + CELL_KEY_SIZE 
	dataSizeEnd := cellPointer + CELL_KEY_SIZE + CELL_DATA_SIZE

	keySize := Endian.Uint16(p.data[cellPointer:cellPointer+CELL_KEY_SIZE])
	dataSize := Endian.Uint16(p.data[dataSizeBegin:dataSizeEnd])
	
	valueBegin := dataSizeEnd + keySize
	value := p.data[valueBegin:valueBegin+dataSize]
	return value 
}

func (p *Page) GetChildAt(idx int) PageId {
	if p.IsLeaf() {
		return 0
	}
	
	var pointer PageId
	
	if idx == 0 {
		pointer = PageId(p.getPointer(PAGE_HEADER_LEFT_CHILD))
	} else {
		cellPointer := p.getCellPointer(idx) 
		childPointerBegin := cellPointer + CELL_KEY_SIZE 
		childPOinterEnd := cellPointer + CELL_KEY_SIZE + CELL_CHILD_POINTER_SIZE

		binary.Decode(p.data[childPointerBegin:childPOinterEnd], Endian, &pointer)
	}
	return pointer 
} 

func (p *Page) PutKeyValue(idx int, key []byte, value []byte) bool {
	if !p.IsLeaf() {
		return false
	}
	
	cell, cellSize := encodeValueCell(key, value)
	p.writeToFreeBlock(idx, cell, cellSize)
	return true 
}


func (p *Page) PutKeyChild(idx int, key []byte, pageId PageId) bool {
	if p.IsLeaf() {
		return false
	}
	
	cell, cellSize := encodePointerCell(key, pageId)
	return p.writeToFreeBlock(idx, cell, cellSize)
}

func (p *Page) writeToFreeBlock(idx int, cell []byte, cellSize uint16) bool {

	payloadSize := cellSize + CELL_POINTER_SIZE
	
	if payloadSize > p.GetFreeSpace() {
		return false 
	} 

	if payloadSize > p.GetFreeBlockSize() && payloadSize < p.GetFreeSpace() {
		p.defrag() 
	}
	
	freeBlockPointer := p.GetFreeBlockOffset()
	cellOffset := freeBlockPointer - cellSize
	keys := p.GetKeys()
	keysSize := keys * CELL_POINTER_SIZE
	keyOffeset := PAGE_HEADER_SIZE + uint16(idx)*CELL_POINTER_SIZE


	copy(p.data[keyOffeset+CELL_POINTER_SIZE:PAGE_HEADER_SIZE+keysSize+CELL_POINTER_SIZE], p.data[keyOffeset:PAGE_HEADER_SIZE+keysSize])
	Endian.PutUint16(p.data[keyOffeset:keyOffeset+CELL_POINTER_SIZE], cellOffset)
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
	keysSize := int(p.GetKeys())
	isLeaf := p.IsLeaf()

	var freeBlockOffset = uint16(PAGE_SIZE)

	for i := range keysSize {
		cellPointer := p.getCellPointer(i)
		var cellSize uint16
		if isLeaf {
			keySize := Endian.Uint16(p.data[cellPointer:cellPointer+CELL_KEY_SIZE])
			dataSize := Endian.Uint16(p.data[cellPointer+CELL_KEY_SIZE:cellPointer+CELL_KEY_SIZE+CELL_DATA_SIZE])
			cellSize = CELL_KEY_SIZE + CELL_DATA_SIZE + keySize + dataSize
		} else {

		}
		
		copy(buffer[freeBlockOffset-cellSize:freeBlockOffset], p.data[cellPointer:cellPointer+cellSize])
		freeBlockOffset -= cellSize
		ptrOffset := PAGE_HEADER_SIZE + (i * CELL_POINTER_SIZE)
		Endian.PutUint16(buffer[ptrOffset:ptrOffset+CELL_POINTER_SIZE], freeBlockOffset)
	}

	copy(p.data, buffer) 
	p.SetFragmented(0)
}

func (p *Page) removeIdx(idx int) []byte {
	return nil	
}



func encodeValueCell(key []byte, data []byte) ([]byte, uint16) {
	dataSizeOffset := CELL_KEY_SIZE
	keyOffset := uint16(CELL_KEY_SIZE+CELL_DATA_SIZE)
	keySize := uint16(len(key)) 
	dataOffset := keyOffset + keySize
	dataSize := uint16(len(data)) 

	cellSize := CELL_KEY_SIZE + len(key) + CELL_DATA_SIZE + len(data)

	buffer := make([]byte, cellSize)
	Endian.PutUint16(buffer[0:CELL_KEY_SIZE], keySize)
	Endian.PutUint16(buffer[dataSizeOffset:dataSizeOffset+CELL_DATA_SIZE], dataSize)
	copy(buffer[keyOffset:keyOffset+keySize], key[0:keySize])
	copy(buffer[dataOffset:dataOffset+dataSize], data)
	
	return buffer, uint16(cellSize) 
}

func encodePointerCell(key []byte, pointer PageId) ([]byte, uint16) {
	keySize := uint16(len(key))
	cellSize := CELL_KEY_SIZE + CELL_CHILD_POINTER_SIZE + keySize
	
	buffer := make([]byte, cellSize)
	Endian.PutUint16(buffer[0:CELL_KEY_SIZE], keySize)
	Endian.PutUint64(buffer[CELL_KEY_SIZE:CELL_KEY_SIZE+CELL_CHILD_POINTER_SIZE], uint64(pointer))
	copy(buffer[CELL_KEY_SIZE+CELL_CHILD_POINTER_SIZE:], key)

	return buffer, cellSize
}

func (p *Page) getCellPointer(idx int) uint16 {
	begin := PAGE_HEADER_SIZE + CELL_POINTER_SIZE*idx 
	end := begin + CELL_POINTER_SIZE
	return Endian.Uint16(p.data[begin:end])
} 

func (p *Page) getPointer(offset int) PageId {
	return PageId(Endian.Uint64(p.data[offset:offset+SZ_PAGE_POINTER]))
}

func (p *Page) getHeaderValue(begin int, offset int) uint16 {
	return Endian.Uint16(p.data[begin:begin+offset])

}


