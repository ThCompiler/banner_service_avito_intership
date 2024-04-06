package slices

// Map applies function to every item of an iterable and returns a slice of the results.
// The source slice and the result may have different types of stored objects.
//
// Example:
//
//	import (
//
//	coreSlices "slices"
//
//	)
//
//	func main() {
//		a := Map([]float64{1.2, 2.3}, func(a float64) int64 { return int64(a) })
//		b := []int64{1, 2}
//		fmt.Println(coreSlices.Equal(a, b))
//	}
//
// Output:
//
//	true
func Map[T, M any](a []T, f func(T) M) []M {
	n := make([]M, len(a))
	for i, e := range a {
		n[i] = f(e)
	}

	return n
}
