package backend

import (
	"bytes"
	"fmt"
	"testing"
)

func TestPageHeaderFlags(t *testing.T) {
	page := MakePage(1, true)

	if !page.IsLeaf() {
		t.Fatalf("expected page to be leaf")
	}
	if page.IsRoot() {
		t.Fatalf("expected page to not be root initially")
	}
	if page.HasOverflow() || page.IsOverflow() {
		t.Fatalf("expected page to not have overflow flags set")
	}

	page.SetRoot(true)
	if !page.IsRoot() || !page.IsLeaf() {
		t.Fatalf("setting root should preserve leaf flag")
	}

	page.SetHasOverflow(true)
	if !page.HasOverflow() || !page.IsRoot() || !page.IsLeaf() {
		t.Fatalf("setting hasOverflow should preserve other flags")
	}

	page.SetIsOverflow(true)
	if !page.IsOverflow() || !page.HasOverflow() || !page.IsRoot() || !page.IsLeaf() {
		t.Fatalf("setting isOverflow should preserve other flags")
	}

	page.SetRoot(false)
	if page.IsRoot() || !page.IsLeaf() || !page.HasOverflow() || !page.IsOverflow() {
		t.Fatalf("clearing root should only clear root bit")
	}

	page.SetParent(42)
	if page.GetParent() != 42 {
		t.Fatalf("expected parent 42, got %d", page.GetParent())
	}

	page.SetLeftSibbling(10)
	page.SetRightSibbling(20)
	if page.GetLeftSibbling() != 10 || page.GetRightSibbling() != 20 {
		t.Fatalf("sibling getters/setters failed")
	}
}

func TestLeafPageInsertAndSearch(t *testing.T) {
	page := MakePage(1, true)

	items := []struct {
		k string
		v string
	}{
		{"cat", "meow"},
		{"apple", "fruit"},
		{"dog", "bark"},
		{"banana", "yellow"},
	}

	for _, item := range items {
		key := []byte(item.k)
		val := []byte(item.v)

		idx, found := page.Search(key)
		if found {
			t.Fatalf("key %s should not be found before insertion", item.k)
		}

		ok := page.PutKeyValue(idx, key, val)
		if !ok {
			t.Fatalf("failed to insert key %s", item.k)
		}
	}

	if page.GetKeys() != 4 {
		t.Fatalf("expected 4 keys, got %d", page.GetKeys())
	}

	expectedKeys := []string{"apple", "banana", "cat", "dog"}
	expectedVals := []string{"fruit", "yellow", "meow", "bark"}

	for i, expK := range expectedKeys {
		k := string(page.GetKeyAt(i))
		v := string(page.GetValueAt(i))

		if k != expK || v != expectedVals[i] {
			t.Fatalf("idx %d: expected (%s, %s), got (%s, %s)", i, expK, expectedVals[i], k, v)
		}

		idx, found := page.Search([]byte(expK))
		if !found || idx != i {
			t.Fatalf("search for %s expected idx %d, got (%d, %v)", expK, i, idx, found)
		}
	}

	// Search for non-existent key between "banana" and "cat"
	idx, found := page.Search([]byte("bear"))
	if found || idx != 2 {
		t.Fatalf("search for bear expected idx 2, false; got (%d, %v)", idx, found)
	}
}

func TestInternalPageInsertAndChildPointers(t *testing.T) {
	page := MakePage(2, false)

	page.SetLeftChild(100)

	// Insert "m" -> child 101
	idx, _ := page.Search([]byte("m"))
	if !page.PutKeyChild(idx, []byte("m"), 101) {
		t.Fatalf("failed to insert key m")
	}

	// Insert "f" -> child 102
	idx, _ = page.Search([]byte("f"))
	if !page.PutKeyChild(idx, []byte("f"), 102) {
		t.Fatalf("failed to insert key f")
	}

	// Insert "t" -> child 103
	idx, _ = page.Search([]byte("t"))
	if !page.PutKeyChild(idx, []byte("t"), 103) {
		t.Fatalf("failed to insert key t")
	}

	if page.GetKeys() != 3 {
		t.Fatalf("expected 3 keys, got %d", page.GetKeys())
	}

	expectedKeys := []string{"f", "m", "t"}
	expectedChildren := []PageId{100, 102, 101, 103}

	for i, expK := range expectedKeys {
		k := string(page.GetKeyAt(i))
		if k != expK {
			t.Fatalf("key at %d: expected %s, got %s", i, expK, k)
		}
	}

	for i, expC := range expectedChildren {
		c := page.GetChildAt(i)
		if c != expC {
			t.Fatalf("child at %d: expected %d, got %d", i, expC, c)
		}
	}
}

func TestPageRemovalAndDefrag(t *testing.T) {
	page := MakePage(1, true)

	for i := 0; i < 10; i++ {
		k := []byte(fmt.Sprintf("key_%02d", i))
		v := []byte(fmt.Sprintf("val_%02d", i))
		idx, _ := page.Search(k)
		page.PutKeyValue(idx, k, v)
	}

	if page.GetKeys() != 10 {
		t.Fatalf("expected 10 keys")
	}

	// Remove index 3 and index 5
	remK3, remV3, err := page.RemoveValueAtIdx(3)
	if err != nil || string(remK3) != "key_03" || string(remV3) != "val_03" {
		t.Fatalf("failed to remove index 3: %v", err)
	}

	if page.GetKeys() != 9 {
		t.Fatalf("expected 9 keys after removal")
	}

	frag := page.GetFragmented()
	if frag == 0 {
		t.Fatalf("expected non-zero fragmented space after removal")
	}

	// Force defragmentation by allocating large dummy payloads until defrag triggers
	initialFreeBlock := page.GetFreeBlockSize()
	page.defrag()

	if page.GetFragmented() != 0 {
		t.Fatalf("expected 0 fragmented space after defrag")
	}
	if page.GetFreeBlockSize() <= initialFreeBlock {
		t.Fatalf("free block size should increase after defrag")
	}

	// Verify remaining keys are intact and correctly ordered
	for i := 0; i < page.GetKeys(); i++ {
		k := string(page.GetKeyAt(i))
		if k == "key_03" {
			t.Fatalf("removed key key_03 should not exist")
		}
	}
}

func TestLeafPageSplitAndMerge(t *testing.T) {
	p1 := MakePage(1, true)
	p2 := MakePage(2, true)

	for i := 0; i < 6; i++ {
		k := []byte(fmt.Sprintf("k%d", i))
		v := []byte(fmt.Sprintf("v%d", i))
		idx, _ := p1.Search(k)
		p1.PutKeyValue(idx, k, v)
	}

	splitKey, err := p1.Split(&p2)
	if err != nil {
		t.Fatalf("split failed: %v", err)
	}

	if string(splitKey) != "k3" {
		t.Fatalf("expected split key k3, got %s", string(splitKey))
	}

	if p1.GetKeys() != 3 || p2.GetKeys() != 3 {
		t.Fatalf("expected 3 keys in each page after split, got %d and %d", p1.GetKeys(), p2.GetKeys())
	}

	if p1.GetRightSibbling() != p2.Id || p2.GetLeftSibbling() != p1.Id {
		t.Fatalf("sibling pointers incorrect after split")
	}

	// Merge p2 back into p1
	err = p1.Merge(&p2, nil)
	if err != nil {
		t.Fatalf("merge failed: %v", err)
	}

	if p1.GetKeys() != 6 {
		t.Fatalf("expected 6 keys after merge, got %d", p1.GetKeys())
	}

	for i := 0; i < 6; i++ {
		expK := fmt.Sprintf("k%d", i)
		if string(p1.GetKeyAt(i)) != expK {
			t.Fatalf("key mismatch at %d: expected %s, got %s", i, expK, string(p1.GetKeyAt(i)))
		}
	}
}

func TestInternalPageSplitAndMerge(t *testing.T) {
	p1 := MakePage(1, false)
	p2 := MakePage(2, false)

	p1.SetLeftChild(10)

	for i := 0; i < 5; i++ {
		k := []byte(fmt.Sprintf("k%d", i))
		c := PageId(20 + i)
		idx, _ := p1.Search(k)
		p1.PutKeyChild(idx, k, c)
	}

	splitKey, err := p1.Split(&p2)
	if err != nil {
		t.Fatalf("internal split failed: %v", err)
	}

	if string(splitKey) != "k2" {
		t.Fatalf("expected split key k2, got %s", string(splitKey))
	}

	// p1 should have 2 keys ("k0", "k1")
	if p1.GetKeys() != 2 {
		t.Fatalf("expected 2 keys in p1, got %d", p1.GetKeys())
	}

	// p2 should have 2 keys ("k3", "k4")
	if p2.GetKeys() != 2 {
		t.Fatalf("expected 2 keys in p2, got %d", p2.GetKeys())
	}

	// p2's left child should be child at index 3 (22, which was right child of k2)
	if p2.GetLeftChild() != 22 {
		t.Fatalf("expected p2 left child 22, got %d", p2.GetLeftChild())
	}

	// Merge p2 back into p1 using splitKey
	err = p1.Merge(&p2, splitKey)
	if err != nil {
		t.Fatalf("internal merge failed: %v", err)
	}

	if p1.GetKeys() != 5 {
		t.Fatalf("expected 5 keys after merge, got %d", p1.GetKeys())
	}
}

func TestPageDataSerialization(t *testing.T) {
	p1 := MakePage(1, true)
	p1.PutKeyValue(0, []byte("key"), []byte("val"))

	rawBytes := p1.Data()
	if len(rawBytes) != PAGE_SIZE {
		t.Fatalf("expected data size %d", PAGE_SIZE)
	}

	p2 := MakePage(2, true)
	p2.SetData(rawBytes)

	if !bytes.Equal(p2.GetKeyAt(0), []byte("key")) || !bytes.Equal(p2.GetValueAt(0), []byte("val")) {
		t.Fatalf("data serialization/deserialization failed")
	}
}
