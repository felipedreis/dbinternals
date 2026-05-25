package ds

import "fmt"

func InsertAt[T any](slice []T, value T, index int) []T{
	if index > len(slice) {
		panic(fmt.Sprintf("Trying to insert at index %d outside of the slice %v", index, slice))
	}

	slice = append(slice, value)
	copy(slice[index+1:], slice[index:])
	slice[index] = value 
	return slice 
} 

func Remove[T any](slice []T, index int) ([]T, T) {
	if index >= len(slice) { 
		panic(fmt.Sprintf("Trying to remove at index %d outside of the slice %v", index, slice))
	}

	ret := slice[index]
	copy(slice[index:], slice[index+1:])
	slice = slice[:len(slice)-1]
	return slice, ret
}
