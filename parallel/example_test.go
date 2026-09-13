package parallel_test

import (
	"errors"
	"fmt"
	"log"
	"slices"

	"github.com/HazelnutParadise/insyra/parallel"
)

// The quick start in Docs/parallel.md.
func ExampleGroupUp() {
	sales := []float64{120, 260, 40, 420, 310}

	total := func() float64 {
		sum := 0.0
		for _, v := range sales {
			sum += v
		}
		return sum
	}
	lowHigh := func() (float64, float64) { return slices.Min(sales), slices.Max(sales) }
	count := func() int { return len(sales) }

	results, err := parallel.GroupUp(total, lowHigh, count).Run().AwaitResult()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(results[0][0])
	fmt.Println(results[1][0], results[1][1])
	fmt.Println(results[2][0])
	// Output:
	// 1150
	// 40 420
	// 5
}

func ExampleRunningGroup_AwaitNoResult() {
	sales := []float64{120, 260, 40, 420, 310}
	var low, high float64

	err := parallel.GroupUp(
		func() { low = slices.Min(sales) },
		func() { high = slices.Max(sales) },
	).Run().AwaitNoResult()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(low, high)
	// Output: 40 420
}

func ExampleWorkerError() {
	results, err := parallel.GroupUp(
		func() int { return 1 },
		func() int { panic("out of range") },
	).Run().AwaitResult()

	var we *parallel.WorkerError
	if errors.As(err, &we) {
		fmt.Println("function", we.Index, "panicked:", we.Panic)
	}
	fmt.Println(results[0], results[1] == nil)
	// Output:
	// function 1 panicked: out of range
	// [1] true
}
