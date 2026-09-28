package nn

import (
	"errors"
	"io/fs"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

func TestSequentialMNISTConvergence(t *testing.T) {
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
		t.Fatalf("MNIST shapes are train=%d x %d x %d/%d labels, test=%d x %d x %d/%d labels", trainCount, rows, cols, len(trainLabels), testCount, testRows, testCols, len(testLabels))
	}

	const (
		seed         = int64(20260803)
		batchSize    = 128
		learningRate = float32(1e-3)
	)
	tape := NewTape(seed)
	model, err := NewSequential(tape, Dense(rows*cols, 128), ReLU(), Dense(128, 10))
	if err != nil {
		t.Fatal(err)
	}
	inputSize := rows * cols
	meanLosses := make([]float32, 0, 2)
	accuracies := make([]float32, 0, 2)
	for epoch := 0; epoch < 2; epoch++ {
		order := tape.rng.Perm(trainCount)
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
			// Keep the parameter registry and optimizer state, but release the
			// prior batch graph before recording the next one.
			tape.ops = nil
			tape.grads = make(map[*Tensor]*Tensor)
			logits, err := model.Forward(tape, batchInput)
			if err != nil {
				t.Fatalf("epoch %d batch %d forward: %v", epoch+1, batchCount, err)
			}
			loss, err := tape.SoftmaxCrossEntropy(logits, batchTargets)
			if err != nil {
				t.Fatalf("epoch %d batch %d loss: %v", epoch+1, batchCount, err)
			}
			if err := tape.Backward(loss); err != nil {
				t.Fatalf("epoch %d batch %d backward: %v", epoch+1, batchCount, err)
			}
			if err := tape.Adam(learningRate); err != nil {
				t.Fatalf("epoch %d batch %d Adam: %v", epoch+1, batchCount, err)
			}
			lossTotal += float64(loss.data[0])
			batchCount++
		}
		meanLoss := float32(lossTotal / float64(batchCount))
		meanLosses = append(meanLosses, meanLoss)
		testInput := mustTestTensor(t, []int{testCount, inputSize}, testImages)
		predictionLogits, err := model.Predict(testInput)
		if err != nil {
			t.Fatalf("epoch %d prediction: %v", epoch+1, err)
		}
		predictions := make([]int64, testCount)
		for row := range predictions {
			best := 0
			for classIndex := 1; classIndex < predictionLogits.shape[1]; classIndex++ {
				if predictionLogits.data[row*predictionLogits.shape[1]+classIndex] > predictionLogits.data[row*predictionLogits.shape[1]+best] {
					best = classIndex
				}
			}
			predictions[row] = int64(best)
		}
		correct := 0
		for index, prediction := range predictions {
			if prediction == testLabels[index] {
				correct++
			}
		}
		accuracies = append(accuracies, float32(correct)/float32(testCount))
		t.Logf("epoch %d: mean training loss = %.6f, test accuracy = %.2f%%", epoch+1, meanLoss, 100*accuracies[epoch])
	}

	// The claim is that the sugar changes nothing, so the comparison is against
	// the hand-written tape loop, run here under the same seed, rather than
	// against numbers recorded on one machine: f32 arithmetic differs between
	// platforms, and the pair recorded on arm64 drifted in the fifth digit on
	// amd64 while both paths still agreed with each other.
	handLosses, handAccuracies := handWrittenMNISTRun(t, mnistData{
		trainImages: trainImages,
		trainLabels: trainLabels,
		trainCount:  trainCount,
		testImages:  testImages,
		testLabels:  testLabels,
		testCount:   testCount,
		inputSize:   inputSize,
	}, seed, batchSize, learningRate, len(meanLosses))
	for epoch := range meanLosses {
		if meanLosses[epoch] != handLosses[epoch] {
			t.Fatalf("epoch %d mean loss: Sequential %.6f, hand-written tape %.6f", epoch+1, meanLosses[epoch], handLosses[epoch])
		}
		if accuracies[epoch] != handAccuracies[epoch] {
			t.Fatalf("epoch %d accuracy: Sequential %.2f%%, hand-written tape %.2f%%", epoch+1, 100*accuracies[epoch], 100*handAccuracies[epoch])
		}
	}
	if !(meanLosses[1] < meanLosses[0]/2) {
		t.Fatalf("Sequential MNIST mean losses = %.6f, %.6f; want the second epoch below half the first", meanLosses[0], meanLosses[1])
	}
	if accuracies[1] < 0.95 {
		t.Fatalf("Sequential MNIST final accuracy = %.2f%%, want at least 95%%", 100*accuracies[1])
	}
}

// mnistData is one loaded copy of the dataset, shared by the two runs the
// sugar-changes-nothing comparison needs.
type mnistData struct {
	trainImages []float32
	trainLabels []int64
	trainCount  int
	testImages  []float32
	testLabels  []int64
	testCount   int
	inputSize   int
}

// handWrittenMNISTRun trains the documented hand-written tape loop for the
// given number of epochs and returns its per-epoch mean training loss and test
// accuracy. It is the same loop TestMNISTConvergence runs, stopped at a fixed
// epoch count so the two curves line up element by element.
func handWrittenMNISTRun(t *testing.T, data mnistData, seed int64, batchSize int, learningRate float32, epochs int) (losses, accuracies []float32) {
	t.Helper()
	rng := rand.New(rand.NewSource(seed))
	model := newTestMLP(t, rng, data.inputSize, 128, 10)
	for epoch := 0; epoch < epochs; epoch++ {
		order := rng.Perm(data.trainCount)
		var lossTotal float64
		batchCount := 0
		for start := 0; start < data.trainCount; start += batchSize {
			end := start + batchSize
			if end > data.trainCount {
				end = data.trainCount
			}
			batchRows := end - start
			batchFeatures := make([]float32, batchRows*data.inputSize)
			batchLabels := make([]int64, batchRows)
			for row := 0; row < batchRows; row++ {
				source := order[start+row]
				copy(batchFeatures[row*data.inputSize:(row+1)*data.inputSize], data.trainImages[source*data.inputSize:(source+1)*data.inputSize])
				batchLabels[row] = data.trainLabels[source]
			}
			batchInput := mustTestTensor(t, []int{batchRows, data.inputSize}, batchFeatures)
			batchTargets := mustTestInt64Tensor(t, []int{batchRows}, batchLabels)
			loss, err := model.trainBatch(batchInput, batchTargets, learningRate)
			if err != nil {
				t.Fatalf("hand-written epoch %d batch %d: %v", epoch+1, batchCount, err)
			}
			lossTotal += float64(loss)
			batchCount++
		}
		losses = append(losses, float32(lossTotal/float64(batchCount)))
		testInput := mustTestTensor(t, []int{data.testCount, data.inputSize}, data.testImages)
		predictions, err := model.predict(testInput)
		if err != nil {
			t.Fatalf("hand-written epoch %d prediction: %v", epoch+1, err)
		}
		correct := 0
		for index, prediction := range predictions {
			if prediction == data.testLabels[index] {
				correct++
			}
		}
		accuracies = append(accuracies, float32(correct)/float32(data.testCount))
	}
	return losses, accuracies
}
