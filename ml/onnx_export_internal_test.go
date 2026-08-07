package ml

import (
	"bytes"
	"testing"
)

func TestONNXExplicitZeroIntegerAttributeIsEncoded(t *testing.T) {
	payload := onnxAttrInt("default_int64", 0).marshal()
	if !bytes.Contains(payload, []byte{0x18, 0x00}) {
		t.Fatalf("default_int64 zero is not encoded in %x", payload)
	}
}

func TestONNXScalarInitializerHasScalarShape(t *testing.T) {
	b := &onnxBuilder{}
	b.addInt64ScalarInitializer("category", 0)
	if got := b.graph.Initializers[0].Dims; len(got) != 0 {
		t.Fatalf("scalar initializer dimensions = %v, want scalar", got)
	}
}
