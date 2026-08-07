package nn

import (
	"errors"
	"fmt"
	"io/fs"
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

	if fmt.Sprintf("%.6f", meanLosses[0]) != "0.350281" || fmt.Sprintf("%.6f", meanLosses[1]) != "0.163855" {
		t.Fatalf("Sequential MNIST mean losses = %.6f, %.6f, want 0.350281, 0.163855", meanLosses[0], meanLosses[1])
	}
	if accuracies[1] != float32(0.9584) {
		t.Fatalf("Sequential MNIST final accuracy = %.2f%%, want 95.84%%", 100*accuracies[1])
	}
}
