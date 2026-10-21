package main 

import (
	"fmt"
	
	"github.com/felipedreis/dbinternals/backend"
)

func main() {
	fmt.Println("Hello")
	backend.NewBTree(10)


	left := []int{1, 2, 3, 4}
	right := []int{5, 6, 7, 8}

	fmt.Printf("%v\n", append(left, right...))
	copy(left[2:], right[2:])
	var v int
	left, v = backend.Remove(left, len(left) - 1)
	left = backend.InsertAt(left, 9, len(left) - 1)
	fmt.Println(left)
	fmt.Println(v)
}
