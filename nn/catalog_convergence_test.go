package nn

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// TestSequentialCNNCatalogMNISTConvergence proves the complete catalog on a
// bounded 30,000-row MNIST training subset. The close variant retains the
// second convolution's 14x14 spatial features before the classifier; this is
// necessary for MNIST because global averaging discards digit geometry.
func TestSequentialCNNCatalogMNISTConvergence(t *testing.T) {
	root, ok := os.LookupEnv("INSYRA_NN_MNIST_DIR")
	if !ok || root == "" {
		t.Skip("INSYRA_NN_MNIST_DIR is unset")
	}
	if info, err := os.Stat(root); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			t.Skipf("INSYRA_NN_MNIST_DIR=%q is missing", root)
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
		trainLimit   = 30000
		batchSize    = 128
		learningRate = float32(1e-3)
		maxEpochs    = 8
	)
	if trainCount < trainLimit {
		t.Fatalf("MNIST training rows = %d, want at least %d", trainCount, trainLimit)
	}
	tape := NewTape(20260803)
	model, err := NewSequential(tape,
		Conv2D(1, 8, 3, ConvOptions{Pads: []int{1, 1, 1, 1}}),
		BatchNorm2D(8),
		ReLU(),
		MaxPool2D(2),
		Conv2D(8, 16, 3, ConvOptions{Pads: []int{1, 1, 1, 1}}),
		ReLU(),
		NewFlatten(),
		Dense(16*14*14, 10),
	)
	if err != nil {
		t.Fatal(err)
	}
	meanLosses := make([]float32, 0, maxEpochs)
	accuracies := make([]float32, 0, maxEpochs)
	for epoch := 0; epoch < maxEpochs; epoch++ {
		order := tape.rng.Perm(trainLimit)
		var lossTotal float64
		batchCount := 0
		for start := 0; start < trainLimit; start += batchSize {
			end := start + batchSize
			if end > trainLimit {
				end = trainLimit
			}
			batchRows := end - start
			features := make([]float32, batchRows*rows*cols)
			labels := make([]int64, batchRows)
			for row := 0; row < batchRows; row++ {
				source := order[start+row]
				copy(features[row*rows*cols:(row+1)*rows*cols], trainImages[source*rows*cols:(source+1)*rows*cols])
				labels[row] = trainLabels[source]
			}
			tape.ops = nil
			tape.grads = make(map[*Tensor]*Tensor)
			input := mustTestTensor(t, []int{batchRows, 1, rows, cols}, features)
			target := mustTestInt64Tensor(t, []int{batchRows}, labels)
			logits, err := model.Forward(tape, input)
			if err != nil {
				t.Fatalf("epoch %d batch %d Forward: %v", epoch+1, batchCount, err)
			}
			loss, err := tape.SoftmaxCrossEntropy(logits, target)
			if err != nil {
				t.Fatalf("epoch %d batch %d loss: %v", epoch+1, batchCount, err)
			}
			if err := tape.Backward(loss); err != nil {
				t.Fatalf("epoch %d batch %d Backward: %v", epoch+1, batchCount, err)
			}
			if err := tape.Adam(learningRate); err != nil {
				t.Fatalf("epoch %d batch %d Adam: %v", epoch+1, batchCount, err)
			}
			lossTotal += float64(loss.data[0])
			batchCount++
		}
		meanLoss := float32(lossTotal / float64(batchCount))
		accuracy := catalogMNISTAccuracy(t, model, testImages, testLabels, testCount, rows, cols)
		meanLosses = append(meanLosses, meanLoss)
		accuracies = append(accuracies, accuracy)
		t.Logf("epoch %d: mean training loss = %.6f, test accuracy = %.2f%%", epoch+1, meanLoss, 100*accuracy)
		if accuracy >= 0.97 {
			break
		}
	}
	if len(meanLosses) < 2 || meanLosses[len(meanLosses)-1] >= meanLosses[0]/3 {
		t.Fatalf("CNN loss curve = %v, want a final loss below one third of the first", meanLosses)
	}
	if accuracies[len(accuracies)-1] < 0.97 {
		t.Fatalf("CNN test accuracy curve = %v, want at least 97%%", accuracies)
	}
}

func catalogMNISTAccuracy(t *testing.T, model *Sequential, images []float32, labels []int64, count, rows, cols int) float32 {
	t.Helper()
	const evaluationBatch = 256
	correct := 0
	for start := 0; start < count; start += evaluationBatch {
		end := start + evaluationBatch
		if end > count {
			end = count
		}
		input := mustTestTensor(t, []int{end - start, 1, rows, cols}, images[start*rows*cols:end*rows*cols])
		logits, err := model.Predict(input)
		if err != nil {
			t.Fatalf("CNN Predict batch %d: %v", start/evaluationBatch, err)
		}
		for row := 0; row < end-start; row++ {
			best := 0
			for class := 1; class < logits.shape[1]; class++ {
				if logits.data[row*logits.shape[1]+class] > logits.data[row*logits.shape[1]+best] {
					best = class
				}
			}
			if int64(best) == labels[start+row] {
				correct++
			}
		}
	}
	return float32(correct) / float32(count)
}
