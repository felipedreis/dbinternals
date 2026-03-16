package ds

import (
	"errors"
	"fmt"
	"strconv"
	"testing"
)

type TestKey struct {
	k int
}

func (t TestKey) Compare(other Key) int {
	otherT := other.(TestKey)

	if t.k == otherT.k {
		return 0
	} else if t.k > otherT.k {
		return 1
	} else {
		return -1
	}
}

func (t TestKey) String() string {
	return strconv.Itoa(t.k)

}

func TestBTree_Add(t *testing.T) {
	// 1. Setup: Create a tree with a small nodeSize (e.g., 3 or 4)
	tree := NewBTree(3)

	// 2. Define your test data
	tests := []struct {
		name  string
		key   int
		value string
	}{
		{"First insert, initialize root node", 10, "val10"},
		{"Second insert", 8, "val20"},
		{"Third insert", 5, "val5"},
		{"Splits root node, rot key will be eight", 1, "val1"},
		{"Insert 7 in the left leaf node", 7, "val1"},
		{"Insert 2 in the left leaf node, splits", 2, "val2"},
		{"Inserts 20 in the rightmost node", 20, "val20"},
		{"Inserts 30 in the rightmost node, split it", 30, "val30"},
		{"Inserts 25, saturate the last leaf", 25, "val25"},
		{"Inserts 35, propagate split to the root", 35, "val35"},
		{"Inserts 3 to the leftmost node", 3, "val3"},
		{"Inserts 4 to the leftmost node, split", 4, "val4"},
		{"Inserts 11 ", 11, "val11"},
		{"Inserts 9 splits middle node", 9, "val9"},

		//{"", 1, "val1"},
		//{"", 1, "val1"},
		// Add more here...
	}

	// 3. Execution
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree.Add(TestKey{k: tt.key}, &Value{v: []byte(tt.value)})

			tree.Print()

			if err := validateTree(tree.root); err != nil {

			}

		})
	}
}

func validateTree(node *Node) error {
	if node == nil {
		return nil
	}

	if node.leaf && len(node.child) > 0 {
		return errors.New("Leaf node contain child pointers")
	}

	if !node.leaf && len(node.values) > 0 {
		return errors.New("Internal node should not contain values")
	}

	for i, children := range node.child {
		if children == nil {
			continue
		}
		if children.parent != node {

			errMsg := fmt.Sprintf("Inconsistent node parent relationship at node keys %v and index %d", children, i)
			return errors.New(errMsg)
		}

		if err := validateTree(children); err != nil {
			return err
		}
	}

	return nil
}
