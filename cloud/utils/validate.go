package utils

import "fmt"

// CheckDuplicates alert if item is present more than once in array
func CheckDuplicates[T any](arr []T, fn func(item T) string) error {
	for i := range arr {
		for j := 0; j < i; j++ {
			if fn(arr[i]) == fn(arr[j]) {
				return fmt.Errorf("%s appears multiple times", fn(arr[i]))
			}
		}
	}
	return nil
}
