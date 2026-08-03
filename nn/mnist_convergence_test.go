package nn

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type testMLP struct {
	tape       *Tape
	w1, b1, w2 *Parameter
	b2         *Parameter
}

func readMNISTImages(filename string) (values []float32, count, rows, cols int, err error) {
	payload, err := os.ReadFile(filename)
	if err != nil {
		return nil, 0, 0, 0, fmt.Errorf("%s: %w", filename, err)
	}
	if len(payload) < 16 {
		return nil, 0, 0, 0, fmt.Errorf("%s: truncated IDX image header", filename)
	}
	if magic := binary.BigEndian.Uint32(payload[0:4]); magic != 0x00000803 {
		return nil, 0, 0, 0, fmt.Errorf("%s: wrong IDX image magic 0x%08x", filename, magic)
	}
	count = int(binary.BigEndian.Uint32(payload[4:8]))
	rows = int(binary.BigEndian.Uint32(payload[8:12]))
	cols = int(binary.BigEndian.Uint32(payload[12:16]))
	if count <= 0 || rows <= 0 || cols <= 0 {
		return nil, 0, 0, 0, fmt.Errorf("%s: invalid IDX image dimensions %d x %d x %d", filename, count, rows, cols)
	}
	pixels, ok := checkedTestProduct(count, rows, cols)
	if !ok || len(payload)-16 != pixels {
		return nil, 0, 0, 0, fmt.Errorf("%s: image payload has %d bytes, want %d", filename, len(payload)-16, pixels)
	}
	values = make([]float32, pixels)
	for index, value := range payload[16:] {
		values[index] = float32(value) / 255
	}
	return values, count, rows, cols, nil
}

func readMNISTLabels(filename string) (labels []int64, err error) {
	payload, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", filename, err)
	}
	if len(payload) < 8 {
		return nil, fmt.Errorf("%s: truncated IDX label header", filename)
	}
	if magic := binary.BigEndian.Uint32(payload[0:4]); magic != 0x00000801 {
		return nil, fmt.Errorf("%s: wrong IDX label magic 0x%08x", filename, magic)
	}
	count := int(binary.BigEndian.Uint32(payload[4:8]))
	if count <= 0 || len(payload)-8 != count {
		return nil, fmt.Errorf("%s: label payload has %d bytes, want %d", filename, len(payload)-8, count)
	}
	labels = make([]int64, count)
	for index, value := range payload[8:] {
		labels[index] = int64(value)
	}
	return labels, nil
}

func checkedTestProduct(values ...int) (int, bool) {
	product := 1
	for _, value := range values {
		if value <= 0 || product > int(^uint(0)>>1)/value {
			return 0, false
		}
		product *= value
	}
	return product, true
}

func seededHeWeights(rng *rand.Rand, fanIn, fanOut int) []float32 {
	values := make([]float32, fanIn*fanOut)
	scale := math.Sqrt(2 / float64(fanIn))
	for index := range values {
		values[index] = float32(rng.NormFloat64() * scale)
	}
	return values
}

func newTestMLP(t *testing.T, rng *rand.Rand, inputSize, hiddenSize, classCount int) *testMLP {
	t.Helper()
	tape := NewTape()
	parameter := func(shape []int, data []float32) *Parameter {
		value := mustTestTensor(t, shape, data)
		marked, err := tape.Param(value)
		if err != nil {
			t.Fatal(err)
		}
		return marked
	}
	return &testMLP{
		tape: tape,
		w1:   parameter([]int{inputSize, hiddenSize}, seededHeWeights(rng, inputSize, hiddenSize)),
		b1:   parameter([]int{hiddenSize}, make([]float32, hiddenSize)),
		w2:   parameter([]int{hiddenSize, classCount}, seededHeWeights(rng, hiddenSize, classCount)),
		b2:   parameter([]int{classCount}, make([]float32, classCount)),
	}
}

func (m *testMLP) resetTape() {
	// Keep Parameter values and their Adam moments, but release the previous
	// batch's graph and tensors before the next batch is recorded.
	m.tape.ops = nil
	m.tape.grads = make(map[*Tensor]*Tensor)
}

func (m *testMLP) trainBatch(input, labels *Tensor, learningRate float32) (float32, error) {
	m.resetTape()
	hidden, err := m.tape.MatMul(input, m.w1.Value())
	if err != nil {
		return 0, err
	}
	hidden, err = m.tape.Add(hidden, m.b1.Value())
	if err != nil {
		return 0, err
	}
	hidden, err = m.tape.Relu(hidden)
	if err != nil {
		return 0, err
	}
	logits, err := m.tape.MatMul(hidden, m.w2.Value())
	if err != nil {
		return 0, err
	}
	logits, err = m.tape.Add(logits, m.b2.Value())
	if err != nil {
		return 0, err
	}
	loss, err := m.tape.SoftmaxCrossEntropy(logits, labels)
	if err != nil {
		return 0, err
	}
	if err := m.tape.Backward(loss); err != nil {
		return 0, err
	}
	if err := m.tape.Adam(learningRate); err != nil {
		return 0, err
	}
	return loss.data[0], nil
}

func (m *testMLP) predict(input *Tensor) ([]int64, error) {
	hidden, err := MatMul(input, m.w1.Value())
	if err != nil {
		return nil, err
	}
	hidden, err = Add(hidden, m.b1.Value())
	if err != nil {
		return nil, err
	}
	hidden, err = Relu(hidden)
	if err != nil {
		return nil, err
	}
	logits, err := MatMul(hidden, m.w2.Value())
	if err != nil {
		return nil, err
	}
	logits, err = Add(logits, m.b2.Value())
	if err != nil {
		return nil, err
	}
	values := logits.data
	classes := logits.shape[len(logits.shape)-1]
	predictions := make([]int64, logits.shape[0])
	for row := range predictions {
		best := 0
		for classIndex := 1; classIndex < classes; classIndex++ {
			if values[row*classes+classIndex] > values[row*classes+best] {
				best = classIndex
			}
		}
		predictions[row] = int64(best)
	}
	return predictions, nil
}

func TestMNISTIDXReader(t *testing.T) {
	dir := t.TempDir()
	imageFile := filepath.Join(dir, "images.idx")
	labelFile := filepath.Join(dir, "labels.idx")
	imagePayload := make([]byte, 16+3)
	binary.BigEndian.PutUint32(imagePayload[0:4], 0x00000803)
	binary.BigEndian.PutUint32(imagePayload[4:8], 1)
	binary.BigEndian.PutUint32(imagePayload[8:12], 1)
	binary.BigEndian.PutUint32(imagePayload[12:16], 3)
	imagePayload[16], imagePayload[17], imagePayload[18] = 0, 127, 255
	if err := os.WriteFile(imageFile, imagePayload, 0o600); err != nil {
		t.Fatal(err)
	}
	values, count, rows, cols, err := readMNISTImages(imageFile)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 || rows != 1 || cols != 3 {
		t.Fatalf("image dimensions = %d x %d x %d, want 1 x 1 x 3", count, rows, cols)
	}
	wantValues := []float32{0, float32(127) / 255, 1}
	for index, want := range wantValues {
		if values[index] != want {
			t.Fatalf("image pixel[%d] = %g, want %g", index, values[index], want)
		}
	}

	labelPayload := make([]byte, 8+2)
	binary.BigEndian.PutUint32(labelPayload[0:4], 0x00000801)
	binary.BigEndian.PutUint32(labelPayload[4:8], 2)
	labelPayload[8], labelPayload[9] = 3, 7
	if err := os.WriteFile(labelFile, labelPayload, 0o600); err != nil {
		t.Fatal(err)
	}
	labels, err := readMNISTLabels(labelFile)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := fmt.Sprint(labels), "[3 7]"; got != want {
		t.Fatalf("labels = %s, want %s", got, want)
	}
}

func TestMNISTIDXReaderRejectsMalformedFiles(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name string
		data []byte
		read func(string) error
	}{
		{
			name: "wrong-image-magic.idx",
			data: make([]byte, 16),
			read: func(filename string) error {
				_, _, _, _, err := readMNISTImages(filename)
				return err
			},
		},
		{
			name: "wrong-label-magic.idx",
			data: make([]byte, 8),
			read: func(filename string) error {
				_, err := readMNISTLabels(filename)
				return err
			},
		},
		{
			name: "truncated-images.idx",
			data: []byte{0, 0, 8, 3},
			read: func(filename string) error {
				_, _, _, _, err := readMNISTImages(filename)
				return err
			},
		},
		{
			name: "truncated-labels.idx",
			data: []byte{0, 0, 8, 1, 0, 0, 0, 2, 4},
			read: func(filename string) error {
				_, err := readMNISTLabels(filename)
				return err
			},
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			filename := filepath.Join(dir, testCase.name)
			if err := os.WriteFile(filename, testCase.data, 0o600); err != nil {
				t.Fatal(err)
			}
			err := testCase.read(filename)
			if err == nil {
				t.Fatal("malformed IDX file was accepted")
			}
			if !strings.Contains(err.Error(), filename) {
				t.Fatalf("error %q does not name %q", err, filename)
			}
		})
	}
}

func TestSeededHeInitializationDeterministic(t *testing.T) {
	const seed = int64(20260803)
	first := seededHeWeights(rand.New(rand.NewSource(seed)), 7, 11)
	second := seededHeWeights(rand.New(rand.NewSource(seed)), 7, 11)
	for index := range first {
		if first[index] != second[index] {
			t.Fatalf("weight[%d] differs between fixed-seed runs: %g != %g", index, first[index], second[index])
		}
		if math.IsNaN(float64(first[index])) || math.IsInf(float64(first[index]), 0) {
			t.Fatalf("weight[%d] is not finite: %g", index, first[index])
		}
	}
}

func TestMLPMicroConvergence(t *testing.T) {
	rng := rand.New(rand.NewSource(20260803))
	const sampleCount = 64
	features := make([]float32, sampleCount*2)
	labels := make([]int64, sampleCount)
	for row := 0; row < sampleCount; row++ {
		center := float64(-2)
		if row >= sampleCount/2 {
			center = 2
			labels[row] = 1
		}
		features[2*row] = float32(center + 0.35*rng.NormFloat64())
		features[2*row+1] = float32(center + 0.35*rng.NormFloat64())
	}
	input := mustTestTensor(t, []int{sampleCount, 2}, features)
	targets := mustTestInt64Tensor(t, []int{sampleCount}, labels)
	model := newTestMLP(t, rng, 2, 8, 2)
	const maxSteps = 200
	for step := 0; step < maxSteps; step++ {
		if _, err := model.trainBatch(input, targets, 0.01); err != nil {
			t.Fatalf("step %d: %v", step, err)
		}
		predictions, err := model.predict(input)
		if err != nil {
			t.Fatalf("step %d prediction: %v", step, err)
		}
		correct := 0
		for index, prediction := range predictions {
			if prediction == labels[index] {
				correct++
			}
		}
		if correct == sampleCount {
			return
		}
	}
	t.Fatalf("micro MLP did not reach 100%% accuracy in %d steps", maxSteps)
}

func TestMNISTConvergence(t *testing.T) {
	root, ok := os.LookupEnv("INSYRA_NN_MNIST_DIR")
	if !ok || root == "" {
		t.Skip("INSYRA_NN_MNIST_DIR is unset")
	}
	info, err := os.Stat(root)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			t.Skipf("INSYRA_NN_MNIST_DIR=%q is missing", root)
		}
		t.Fatalf("stat INSYRA_NN_MNIST_DIR=%q: %v", root, err)
	}
	if !info.IsDir() {
		t.Fatalf("INSYRA_NN_MNIST_DIR=%q is not a directory", root)
	}
	imagePath := func(name string) string { return filepath.Join(root, name) }
	trainImages, trainCount, rows, cols, err := readMNISTImages(imagePath("train-images-idx3-ubyte"))
	if err != nil {
		mnistReadFailure(t, root, err)
	}
	trainLabels, err := readMNISTLabels(imagePath("train-labels-idx1-ubyte"))
	if err != nil {
		mnistReadFailure(t, root, err)
	}
	testImages, testCount, testRows, testCols, err := readMNISTImages(imagePath("t10k-images-idx3-ubyte"))
	if err != nil {
		mnistReadFailure(t, root, err)
	}
	testLabels, err := readMNISTLabels(imagePath("t10k-labels-idx1-ubyte"))
	if err != nil {
		mnistReadFailure(t, root, err)
	}
	if rows != 28 || cols != 28 || testRows != rows || testCols != cols || len(trainLabels) != trainCount || len(testLabels) != testCount {
		t.Fatalf("MNIST shapes are train=%d x %d x %d/%d labels, test=%d x %d x %d/%d labels; want 28 x 28 and matching labels", trainCount, rows, cols, len(trainLabels), testCount, testRows, testCols, len(testLabels))
	}

	rng := rand.New(rand.NewSource(20260803))
	model := newTestMLP(t, rng, rows*cols, 128, 10)
	inputSize := rows * cols
	const (
		batchSize    = 128
		learningRate = float32(1e-3)
		maxEpochs    = 5
	)
	initialBatchLoss := float32(0)
	finalMeanLoss := float32(0)
	finalAccuracy := float32(0)
	for epoch := 0; epoch < maxEpochs; epoch++ {
		order := rng.Perm(trainCount)
		var lossTotal float64
		batchCount := 0
		for start := 0; start < trainCount; start += batchSize {
			end := start + batchSize
			if end > trainCount {
				end = trainCount
			}
			batchRows := end - start
			batchFeatures := make([]float32, batchRows*inputSize)
			batchLabels := make([]int64, batchRows)
			for row := 0; row < batchRows; row++ {
				source := order[start+row]
				copy(batchFeatures[row*inputSize:(row+1)*inputSize], trainImages[source*inputSize:(source+1)*inputSize])
				batchLabels[row] = trainLabels[source]
			}
			batchInput := mustTestTensor(t, []int{batchRows, inputSize}, batchFeatures)
			batchTargets := mustTestInt64Tensor(t, []int{batchRows}, batchLabels)
			loss, err := model.trainBatch(batchInput, batchTargets, learningRate)
			if err != nil {
				t.Fatalf("epoch %d batch %d: %v", epoch+1, batchCount, err)
			}
			if epoch == 0 && batchCount == 0 {
				initialBatchLoss = loss
			}
			lossTotal += float64(loss)
			batchCount++
		}
		finalMeanLoss = float32(lossTotal / float64(batchCount))
		testInput := mustTestTensor(t, []int{testCount, inputSize}, testImages)
		predictions, err := model.predict(testInput)
		if err != nil {
			t.Fatalf("epoch %d test prediction: %v", epoch+1, err)
		}
		correct := 0
		for index, prediction := range predictions {
			if prediction == testLabels[index] {
				correct++
			}
		}
		finalAccuracy = float32(correct) / float32(testCount)
		t.Logf("epoch %d: mean training loss = %.6f, test accuracy = %.2f%%", epoch+1, finalMeanLoss, 100*finalAccuracy)
		if finalAccuracy >= 0.95 {
			break
		}
	}
	if finalAccuracy < 0.95 {
		t.Fatalf("MNIST test accuracy = %.2f%%, want at least 95%%", 100*finalAccuracy)
	}
	if !(finalMeanLoss < initialBatchLoss/3) {
		t.Fatalf("final mean training loss = %g, initial batch loss = %g; want final < initial/3", finalMeanLoss, initialBatchLoss)
	}
}

func mnistReadFailure(t *testing.T, root string, err error) {
	t.Helper()
	if errors.Is(err, fs.ErrNotExist) {
		t.Skipf("INSYRA_NN_MNIST_DIR=%q is missing a required IDX file: %v", root, err)
	}
	t.Fatalf("read MNIST from INSYRA_NN_MNIST_DIR=%q: %v", root, err)
}
