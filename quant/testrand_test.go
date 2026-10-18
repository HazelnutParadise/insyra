package quant

import (
	"fmt"
	"math/rand"
)

func newTestRand(seed int64) *rand.Rand { return rand.New(rand.NewSource(seed)) }

// monthlyDate gives a date string per trading row, 21 rows to a month.
func monthlyDate(i int) string {
	month := i / 21
	return fmt.Sprintf("%04d-%02d-%02d", 2000+month/12, month%12+1, i%21+1)
}
