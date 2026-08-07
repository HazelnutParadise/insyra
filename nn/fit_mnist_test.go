package nn

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestSequentialFitMNISTConvergence(t *testing.T) {
	root, ok := os.LookupEnv("INSYRA_NN_MNIST_DIR")
	if !ok || root == "" {
		t.Skip("INSYRA_NN_MNIST_DIR is unset; acceptance run fills the Fit arm")
	}
	if info, err := os.Stat(root); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			t.Skipf("INSYRA_NN_MNIST_DIR=%q is missing; acceptance run fills the Fit arm", root)
		}
		t.Fatalf("stat INSYRA_NN_MNIST_DIR=%q: %v", root, err)
	} else if !info.IsDir() {
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
		seed      = int64(20260803)
		batchSize = 128
		epochs    = 2
	)
	trainX := mustTestTensor(t, []int{trainCount, rows * cols}, trainImages)
	trainY := mustTestInt64Tensor(t, []int{trainCount}, trainLabels)
	model, err := NewSequential(NewTape(seed), Dense(rows*cols, 128), ReLU(), Dense(128, 10))
	if err != nil {
		t.Fatal(err)
	}
	result, err := model.Fit(trainX, trainY, FitConfig{
		Epochs: epochs, BatchSize: batchSize, Seed: seed,
		Optimizer: Adam{Rate: 1e-3}, Loss: CrossEntropy{}, Quiet: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	// These are Fit's own pinned digits, not M21's (0.350281, 0.163855). The
	// M21 loop drew its shuffle permutations from the same stream that had
	// already produced the weight initialization; Fit's documented Seed
	// contract gives shuffling its own stream derived from FitConfig.Seed.
	// Both are deterministic, so their trajectories are equally pinnable but
	// necessarily different. Equivalence with a hand-written loop under Fit's
	// own idiom is proven digit-for-digit in fit_test.go.
	if len(result.TrainLosses) != epochs || fmt.Sprintf("%.6f", result.TrainLosses[0]) != "0.347310" || fmt.Sprintf("%.6f", result.TrainLosses[1]) != "0.165883" {
		t.Fatalf("Fit MNIST mean losses = %.6f, %.6f, want 0.347310, 0.165883", result.TrainLosses[0], result.TrainLosses[1])
	}

	testX := mustTestTensor(t, []int{testCount, rows * cols}, testImages)
	logits, err := model.Predict(testX)
	if err != nil {
		t.Fatal(err)
	}
	correct := 0
	for row, label := range testLabels {
		best := 0
		for class := 1; class < logits.shape[1]; class++ {
			if logits.data[row*logits.shape[1]+class] > logits.data[row*logits.shape[1]+best] {
				best = class
			}
		}
		if int64(best) == label {
			correct++
		}
	}
	accuracy := float32(correct) / float32(testCount)
	// 95.47% is Fit's own pinned, deterministic result; it clears the M21
	// bar (>=95%) under Fit's Seed contract.
	if fmt.Sprintf("%.4f", accuracy) != "0.9547" {
		t.Fatalf("Fit MNIST final accuracy = %.4f, want 0.9547", accuracy)
	}
}
