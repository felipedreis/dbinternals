package main 

import (
	"fmt"
	
	"github.com/felipedreis/dbinternals/ds"
)

func main() {
	fmt.Println("Hello")
	ds.NewBTree(10)


	left := []int{1, 2, 3, 4}
	right := []int{5, 6, 7, 8}
	
	fmt.Printf("%v\n", append(left, right...))
	copy(left[2:], right[2:])
	var v int 
	left, v = ds.Remove(left, len(left) - 1)
	left = ds.InsertAt(left, 9, len(left) - 1)
	fmt.Println(left)
	fmt.Println(v)
}
