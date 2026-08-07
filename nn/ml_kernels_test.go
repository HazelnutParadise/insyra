package nn

import (
	"strings"
	"testing"
)

func TestMLKernelsRejectUnboundedTargetShapeBeforeAllocation(t *testing.T) {
	input, err := NewTensor([]int{1, 2}, []float32{1, 2})
	if err != nil {
		t.Fatal(err)
	}
	_, err = linearRegressor(input, map[string]protoAttribute{
		"coefficients": {floats: []float32{1, 2}},
		"targets":      {intValue: int64(maxInt()), hasInt: true},
	})
	if err == nil || !strings.Contains(err.Error(), "coefficient count overflows") {
		t.Fatalf("linear regressor error = %v, want checked coefficient-count overflow", err)
	}
}

func TestMLClassifiersRejectUnknownPostTransform(t *testing.T) {
	input, err := NewTensor([]int{1, 1}, []float32{1})
	if err != nil {
		t.Fatal(err)
	}
	attributes := map[string]protoAttribute{
		"classlabels_ints": {ints: []int64{0, 1}},
		"coefficients":     {floats: []float32{1}},
		"post_transform":   {string: []byte("BOGUS")},
	}
	if _, _, err := linearClassifier(input, attributes); err == nil || !strings.Contains(err.Error(), "post_transform") {
		t.Fatalf("linear classifier error = %v, want named post_transform refusal", err)
	}
	if _, _, err := treeEnsembleClassifier(input, simpleTreeClassifierAttributesWithTransform("BOGUS")); err == nil || !strings.Contains(err.Error(), "post_transform") {
		t.Fatalf("tree classifier error = %v, want named post_transform refusal", err)
	}
}

func simpleTreeClassifierAttributesWithTransform(transform string) map[string]protoAttribute {
	attributes := simpleTreeClassifierAttributes()
	attributes["post_transform"] = protoAttribute{string: []byte(transform)}
	return attributes
}
