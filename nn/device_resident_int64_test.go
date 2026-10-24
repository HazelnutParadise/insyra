package nn

import (
	"strings"
	"testing"
)

func TestHostKernelsRefuseAResidentInt64Tensor(t *testing.T) {
	device := residentTestDevice(t)
	labels, err := NewInt64Tensor([]int{3}, []int64{0, 1, 2})
	if err != nil {
		t.Fatal(err)
	}
	moved := mustTo(t, labels, device)
	defer moved.Release()
	check := func(name string, err error) {
		t.Helper()
		if err == nil || !strings.Contains(err.Error(), "ToHost") {
			t.Errorf("%s on a resident int64 tensor: error = %v, want one naming ToHost", name, err)
		}
	}
	_, err = Add(moved, moved)
	check("Add", err)
	_, err = Mul(moved, moved)
	check("Mul", err)
	_, err = Pow(moved, moved)
	check("Pow", err)
	_, err = Sub(moved, moved)
	check("Sub", err)

	image, _ := NewTensor([]int{1, 1, 2, 2}, []float32{1, 2, 3, 4})
	sizes, _ := NewInt64Tensor([]int{4}, []int64{1, 1, 4, 4})
	sizesOnDevice := mustTo(t, sizes, device)
	defer sizesOnDevice.Release()
	_, err = Resize(image, nil, sizesOnDevice)
	check("Resize sizes", err)

	boxes, _ := NewTensor([]int{1, 1, 4}, []float32{0, 0, 1, 1})
	scores, _ := NewTensor([]int{1, 1, 1}, []float32{0.9})
	maxOut, _ := NewInt64Tensor([]int{1}, []int64{1})
	maxOutOnDevice := mustTo(t, maxOut, device)
	defer maxOutOnDevice.Release()
	_, err = NonMaxSuppression(boxes, scores, maxOutOnDevice, nil, nil)
	check("NonMaxSuppression", err)

	table, _ := NewTensor([]int{3, 2}, []float32{1, 2, 3, 4, 5, 6})
	logits, _ := NewTensor([]int{3, 3}, []float32{1, 2, 3, 4, 5, 6, 7, 8, 9})
	tape := NewTape()
	defer tape.Close()
	_, err = tape.Embedding(table, moved)
	check("Embedding", err)
	// Host logits with resident labels mix devices, which the tape refuses
	// with its own message rather than reading the absent host slice.
	if _, err = tape.SoftmaxCrossEntropy(logits, moved); err == nil {
		t.Error("SoftmaxCrossEntropy with host logits and resident labels: want an error")
	}
}
