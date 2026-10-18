package insyra

import "testing"

// A 24-byte Big5 file where chardet's recognizers tie. The answer used to
// change from call to call with the order the recognizers finished in.
func TestDetectEncodingOfATieIsStable(t *testing.T) {
	sample := []byte("\xa6W\xba\xd9,\xbc\xc6\xb6q\n\xc4\xab\xaaG,3\n\xad\xbb\xbf\xbc,5\n")
	first, err := detectEncodingOfSample(sample, "big5.csv")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 300; i++ {
		got, err := detectEncodingOfSample(sample, "big5.csv")
		if err != nil || got != first {
			t.Fatalf("call %d: got %q, %v; first call gave %q", i, got, err, first)
		}
	}
}
