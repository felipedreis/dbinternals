package ds

import (
	"errors"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
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
func init() {
	log.SetOutput(os.Stdout)
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
	}

	// 3. Execution
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			key := TestKey{k: tt.key}
			value := Value{v: []byte(tt.value)}
			tree.Add(key, value)
			tree.Print()

			value, error := tree.Find(key)

			if error != nil {
				t.Errorf("Couldn't find value for key %v", key)
			}

			fmt.Printf("Value: %s\n", string(value.v))

			if error = validateTree(tree.root); error != nil {
				t.Errorf("Expected: %s, Error %v\n", tt.name, error)
			}

		})
	}
}

const TREE_SIZE int = 4
const TEST_CASES int = 5 * TREE_SIZE

func TestBTreee_InsertRandomOrder_PreserveProperties(t *testing.T) {
	tree := NewBTree(TREE_SIZE)
	randVals := rand.Perm(TEST_CASES)

	for _, i := range randVals {
		testKey := TestKey{k: i}
		testValue := Value{v: []byte("val" + strconv.Itoa(i))}
		t.Run(fmt.Sprintf("Adding key %d", i), func(t *testing.T) {
			tree.Add(testKey, testValue)
			
			if error := validateTree(tree.root); error != nil {
				t.Errorf("Adding key %v got error %v", testKey, error)
			}
		})
	}

}

func TestBTree_RemoveIncreasingOrder(t *testing.T) {
	tree := getRandomTree(TREE_SIZE, TEST_CASES)
	
	if e := validateTree(tree.root); e != nil {
		t.Errorf("Tree was not constructed correctly: %v", e)
	}

	for i := range TEST_CASES { 
		t.Run("Removing key " + strconv.Itoa(i), func(t *testing.T) {
			testKey := TestKey{k: i}
			removeOneElement(tree, testKey, t)
		})
	}
}

func TestBTree_RemoveDecreasingOder(t *testing.T) {
	log.SetOutput(os.Stdout)
	tree := getRandomTree(TREE_SIZE, TEST_CASES)
	
	if e := validateTree(tree.root); e != nil {
		t.Errorf("Tree was not constructed correctly: %v", e)
	}
	
	for i := TEST_CASES - 1; i >= 0; i-- { 
		t.Run("Removing key " + strconv.Itoa(i), func(t *testing.T) {
			testKey := TestKey{k: i}
			removeOneElement(tree, testKey, t)
		})
	}
}


func TestBTree_RemoveInRandomOrder(t *testing.T) {
	tree := getRandomTree(TREE_SIZE, TEST_CASES)

	if e := validateTree(tree.root); e != nil {
		t.Errorf("Tree was not constructed correctly: %v", e)
	}

	randRemoves := rand.Perm(TEST_CASES)

	for _, i := range randRemoves {
		t.Run("Removing key " + strconv.Itoa(i), func(t *testing.T) {
			testKey := TestKey{k: i}
			removeOneElement(tree, testKey, t)
		})
	}
}

func removeOneElement(tree *InMemoryBTree, testKey Key, t *testing.T) {
	tree.Print()
	_, error := tree.Remove(testKey)

	if error != nil {
		t.Errorf("Expected to remove key %v but got error %v", testKey, error)
	}

	if error := validateTree(tree.root); error != nil {
		t.Errorf("Removing key %v got error %v", testKey, error)
	}
}

func getRandomTree(nodeSize int, elements int) *InMemoryBTree {
	tree := NewBTree(nodeSize)
	randVals := rand.Perm(elements)
	for _, i := range randVals {
		testKey := TestKey{k: i}
		testValue := Value{v: []byte("val" + strconv.Itoa(i))}
		tree.Add(testKey, testValue)
	}

	tree.Print()

	return tree
}


func validateTree(node *Node) error {
	if node == nil {
		return nil
	}

	if node.Leaf && len(node.Child) > 0 {
		return errors.New("Leaf node contain child pointers")
	}

	if !node.Leaf && len(node.Values) > 0 {
		return errors.New("Internal node should not contain values")
	}

	if !node.Leaf  && (node.Right != nil || node.Left != nil) {
		return fmt.Errorf("Internal nodes %v should not have sibblings relationships", node)
	}

	if !node.Leaf &&  len(node.Child) != len(node.Keys) + 1 {
		return fmt.Errorf("Node %v has %d keys and %d child, which is invalid", 
			node, len(node.Keys), len(node.Child))
	}

	for i, key := range node.Keys {
		next := i + 1

		if next < len(node.Keys) && key.Compare(node.Keys[next]) >= 0 {
			return fmt.Errorf("Node %v has keys outside of order at position %d,%d", node, i, next)
			
		}
	}

	for i, children := range node.Child {
		if children == nil {
			continue
		}
		if children.Parent != node {
			errMsg := fmt.Sprintf("Inconsistent node parent relationship at node.Keys %v and index %d", children, i)
			return errors.New(errMsg)
		}

		if children.Leaf {
			var prev *Node
			var next *Node 

			if i > 0 {
				prev = node.Child[i-1] 
			} 

			if i < len(node.Child)-1 {
				next = node.Child[i + 1]
			}
			if i == 0 && children.Left != nil && children.Left.Parent == node {
				return fmt.Errorf(`Inconsistent left children at first node %v. Expected nil 
						or belonging to another parent %v`,
					children, children.Left)
			}

			if i == len(node.Child)-1 && children.Right != nil && children.Right.Parent == node {
				return fmt.Errorf("Insconsistent right relation at last node  %v. Expected nil found %v",
					children, children.Right)
			}

			if i > 0 && children.Left != prev { 
				return fmt.Errorf("Insconsistent left relation at node  %v idx %d " +
				 "children %v left found was %v but expected was %v", node, i, children, prev, children.Left)
			 }
			
			 if i < len(node.Child)-1 && children.Right != next {
				return fmt.Errorf("Insconsistent right relation at node  %v idx %d " + 
				 "children %v left found was %v but expected was %v", node, i, children, next, children.Right)
			}
		} 

		if err := validateTree(children); err != nil {
			return err
		}
	}

	return nil
}
