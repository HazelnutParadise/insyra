package nn

import (
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"time"

	insyra "github.com/HazelnutParadise/insyra"
)

// OptimizerSpec selects one of the optimizers already implemented by Tape.
// Fit deliberately exposes only those tape-level choices in v1.
type OptimizerSpec interface {
	fitOptimizerName() string
	fitOptimizerValidate() error
	fitOptimizerStep(*Tape) error
}

// SGD selects Tape.SGD.
type SGD struct {
	Rate float32
}

func (SGD) fitOptimizerName() string { return "SGD" }

func (o SGD) fitOptimizerValidate() error {
	if math.IsNaN(float64(o.Rate)) || math.IsInf(float64(o.Rate), 0) || o.Rate < 0 {
		return fmt.Errorf("fit config optimizer SGD rate must be finite and non-negative")
	}
	return nil
}

func (o SGD) fitOptimizerStep(tape *Tape) error { return tape.SGD(o.Rate) }

// SGDMomentum selects Tape.SGDMomentum.
type SGDMomentum struct {
	Rate     float32
	Momentum float32
}

func (SGDMomentum) fitOptimizerName() string { return "SGDMomentum" }

func (o SGDMomentum) fitOptimizerValidate() error {
	if math.IsNaN(float64(o.Rate)) || math.IsInf(float64(o.Rate), 0) || o.Rate < 0 {
		return fmt.Errorf("fit config optimizer SGDMomentum rate must be finite and non-negative")
	}
	if math.IsNaN(float64(o.Momentum)) || math.IsInf(float64(o.Momentum), 0) || o.Momentum < 0 {
		return fmt.Errorf("fit config optimizer SGDMomentum momentum must be finite and non-negative")
	}
	return nil
}

func (o SGDMomentum) fitOptimizerStep(tape *Tape) error {
	return tape.SGDMomentum(o.Rate, o.Momentum)
}

// Adam selects Tape.Adam.
type Adam struct {
	Rate float32
}

func (Adam) fitOptimizerName() string { return "Adam" }

func (o Adam) fitOptimizerValidate() error {
	if math.IsNaN(float64(o.Rate)) || math.IsInf(float64(o.Rate), 0) || o.Rate < 0 {
		return fmt.Errorf("fit config optimizer Adam rate must be finite and non-negative")
	}
	return nil
}

func (o Adam) fitOptimizerStep(tape *Tape) error { return tape.Adam(o.Rate) }

// AdamW selects Tape.AdamW.
type AdamW struct {
	Rate        float32
	WeightDecay float32
}

func (AdamW) fitOptimizerName() string { return "AdamW" }

func (o AdamW) fitOptimizerValidate() error {
	if math.IsNaN(float64(o.Rate)) || math.IsInf(float64(o.Rate), 0) || o.Rate < 0 {
		return fmt.Errorf("fit config optimizer AdamW rate must be finite and non-negative")
	}
	if math.IsNaN(float64(o.WeightDecay)) || math.IsInf(float64(o.WeightDecay), 0) || o.WeightDecay < 0 {
		return fmt.Errorf("fit config optimizer AdamW weight decay must be finite and non-negative")
	}
	return nil
}

func (o AdamW) fitOptimizerStep(tape *Tape) error { return tape.AdamW(o.Rate, o.WeightDecay) }

// LossSpec selects one of the losses already implemented by Tape.
type LossSpec interface {
	fitLossName() string
	fitLossValidate(*Tensor, *Tensor) error
	fitLoss(*Tape, *Tensor, *Tensor) (*Tensor, error)
}

// CrossEntropy selects Tape.SoftmaxCrossEntropy.
type CrossEntropy struct{}

func (CrossEntropy) fitLossName() string { return "CrossEntropy" }

func (CrossEntropy) fitLossValidate(prediction, target *Tensor) error {
	if target == nil || target.dtype != DTypeInt64 {
		return fmt.Errorf("fit config loss CrossEntropy requires int64 targets")
	}
	if prediction == nil || len(prediction.shape) != 2 || len(target.shape) != 1 || prediction.shape[0] != target.shape[0] {
		return fmt.Errorf("fit config loss CrossEntropy requires logits [N,C] and labels [N]")
	}
	return nil
}

func (CrossEntropy) fitLoss(tape *Tape, prediction, target *Tensor) (*Tensor, error) {
	return tape.SoftmaxCrossEntropy(prediction, target)
}

// MSE selects Tape.MSELoss.
type MSE struct{}

func (MSE) fitLossName() string { return "MSE" }

func (MSE) fitLossValidate(prediction, target *Tensor) error {
	if target == nil || target.dtype != DTypeFloat32 {
		return fmt.Errorf("fit config loss MSE requires float32 targets")
	}
	if prediction == nil || !sameShape(prediction.shape, target.shape) {
		return fmt.Errorf("fit config loss MSE requires prediction and target shapes to match")
	}
	return nil
}

func (MSE) fitLoss(tape *Tape, prediction, target *Tensor) (*Tensor, error) {
	return tape.MSELoss(prediction, target)
}

// BCEWithLogits selects Tape.BCEWithLogitsLoss.
type BCEWithLogits struct{}

func (BCEWithLogits) fitLossName() string { return "BCEWithLogits" }

func (BCEWithLogits) fitLossValidate(prediction, target *Tensor) error {
	if target == nil || target.dtype != DTypeFloat32 {
		return fmt.Errorf("fit config loss BCEWithLogits requires float32 targets")
	}
	if prediction == nil || !sameShape(prediction.shape, target.shape) {
		return fmt.Errorf("fit config loss BCEWithLogits requires prediction and target shapes to match")
	}
	return nil
}

func (BCEWithLogits) fitLoss(tape *Tape, prediction, target *Tensor) (*Tensor, error) {
	return tape.BCEWithLogitsLoss(prediction, target)
}

// These aliases keep selector names close to the corresponding tape methods.
type SoftmaxCrossEntropy = CrossEntropy
type MSELoss = MSE
type BCEWithLogitsLoss = BCEWithLogits

// FitConfig controls one complete Sequential training run.
type FitConfig struct {
	Epochs    int
	BatchSize int
	Seed      int64
	NoShuffle bool
	Optimizer OptimizerSpec
	Loss      LossSpec
	ValX      *Tensor
	ValY      *Tensor
	Progress  func(FitEpoch)
	Quiet     bool
}

// FitEpoch is the progress payload for one completed epoch. ValLoss is valid
// only when HasValLoss is true.
type FitEpoch struct {
	Epoch         int
	Epochs        int
	TrainLoss     float64
	ValLoss       float64
	HasValLoss    bool
	Elapsed       time.Duration
	RowsPerSecond float64
}

// FitResult contains the losses and timings from every completed epoch.
type FitResult struct {
	TrainLosses []float64
	ValLosses   []float64
	Epochs      []FitEpoch
	Elapsed     time.Duration
}

// Fit trains the Sequential using the existing tape forward, loss, backward,
// and optimizer methods. It is intentionally a thin, deterministic loop.
func (s *Sequential) Fit(x, y *Tensor, cfg FitConfig) (*FitResult, error) {
	if s == nil {
		return nil, fmt.Errorf("fit sequential is nil")
	}
	if s.tape == nil {
		return nil, fmt.Errorf("fit sequential tape is nil")
	}
	if err := validateFitConfig(x, y, cfg); err != nil {
		return nil, err
	}
	if err := validateFitRows(x, y, "training"); err != nil {
		return nil, err
	}
	if cfg.ValX != nil || cfg.ValY != nil {
		if err := validateFitRows(cfg.ValX, cfg.ValY, "validation"); err != nil {
			return nil, err
		}
	}

	// Keep the model's tape and parameter registry so optimizer state survives
	// batches. Both random streams are owned by this call and derive from Seed.
	shuffleRNG := rand.New(rand.NewSource(cfg.Seed))
	s.tape.rng = rand.New(rand.NewSource(cfg.Seed))
	result := &FitResult{
		TrainLosses: make([]float64, 0, cfg.Epochs),
		Epochs:      make([]FitEpoch, 0, cfg.Epochs),
	}
	runStarted := time.Now()
	rows := x.shape[0]
	for epoch := 0; epoch < cfg.Epochs; epoch++ {
		epochStarted := time.Now()
		order := fitOrder(shuffleRNG, rows, cfg.NoShuffle)
		var lossTotal float64
		batchCount := 0
		for start := 0; start < rows; start += cfg.BatchSize {
			end := start + cfg.BatchSize
			if end > rows {
				end = rows
			}
			batchX, err := fitBatch(x, order[start:end])
			if err != nil {
				return nil, fmt.Errorf("fit epoch %d batch %d input: %w", epoch+1, batchCount+1, err)
			}
			batchY, err := fitBatch(y, order[start:end])
			if err != nil {
				return nil, fmt.Errorf("fit epoch %d batch %d target: %w", epoch+1, batchCount+1, err)
			}
			s.tape.ops = nil
			s.tape.grads = make(map[*Tensor]*Tensor)
			prediction, err := s.Forward(s.tape, batchX)
			if err != nil {
				return nil, fmt.Errorf("fit epoch %d batch %d forward: %w", epoch+1, batchCount+1, err)
			}
			if err := cfg.Loss.fitLossValidate(prediction, batchY); err != nil {
				return nil, fmt.Errorf("fit epoch %d batch %d %s: %w", epoch+1, batchCount+1, cfg.Loss.fitLossName(), err)
			}
			loss, err := cfg.Loss.fitLoss(s.tape, prediction, batchY)
			if err != nil {
				return nil, fmt.Errorf("fit epoch %d batch %d loss: %w", epoch+1, batchCount+1, err)
			}
			if err := s.tape.Backward(loss); err != nil {
				return nil, fmt.Errorf("fit epoch %d batch %d backward: %w", epoch+1, batchCount+1, err)
			}
			if err := cfg.Optimizer.fitOptimizerStep(s.tape); err != nil {
				return nil, fmt.Errorf("fit epoch %d batch %d optimizer %s: %w", epoch+1, batchCount+1, cfg.Optimizer.fitOptimizerName(), err)
			}
			lossTotal += float64(loss.data[0])
			batchCount++
		}

		progress := FitEpoch{
			Epoch:     epoch + 1,
			Epochs:    cfg.Epochs,
			TrainLoss: lossTotal / float64(batchCount),
			Elapsed:   time.Since(epochStarted),
		}
		if progress.Elapsed > 0 {
			progress.RowsPerSecond = float64(rows) / progress.Elapsed.Seconds()
		}
		if cfg.ValX != nil {
			prediction, err := s.Predict(cfg.ValX)
			if err != nil {
				return nil, fmt.Errorf("fit epoch %d validation predict: %w", epoch+1, err)
			}
			validationTape := NewTape()
			if err := cfg.Loss.fitLossValidate(prediction, cfg.ValY); err != nil {
				return nil, fmt.Errorf("fit epoch %d validation %s: %w", epoch+1, cfg.Loss.fitLossName(), err)
			}
			validationLoss, err := cfg.Loss.fitLoss(validationTape, prediction, cfg.ValY)
			if err != nil {
				return nil, fmt.Errorf("fit epoch %d validation loss: %w", epoch+1, err)
			}
			progress.ValLoss = float64(validationLoss.data[0])
			progress.HasValLoss = true
			result.ValLosses = append(result.ValLosses, progress.ValLoss)
		}
		result.TrainLosses = append(result.TrainLosses, progress.TrainLoss)
		result.Epochs = append(result.Epochs, progress)
		if !cfg.Quiet {
			if progress.HasValLoss {
				insyra.LogInfo("nn", "Sequential.Fit", "epoch %d/%d train_loss=%.12g val_loss=%.12g elapsed=%s rows/s=%.2f", progress.Epoch, progress.Epochs, progress.TrainLoss, progress.ValLoss, progress.Elapsed.Round(time.Millisecond), progress.RowsPerSecond)
			} else {
				insyra.LogInfo("nn", "Sequential.Fit", "epoch %d/%d train_loss=%.12g elapsed=%s rows/s=%.2f", progress.Epoch, progress.Epochs, progress.TrainLoss, progress.Elapsed.Round(time.Millisecond), progress.RowsPerSecond)
			}
		}
		if cfg.Progress != nil {
			cfg.Progress(progress)
		}
	}
	result.Elapsed = time.Since(runStarted)
	return result, nil
}

func validateFitConfig(x, y *Tensor, cfg FitConfig) error {
	if cfg.Epochs <= 0 {
		return fmt.Errorf("fit config Epochs must be positive")
	}
	if cfg.BatchSize <= 0 {
		return fmt.Errorf("fit config BatchSize must be positive")
	}
	if isNilFitInterface(cfg.Optimizer) {
		return fmt.Errorf("fit config Optimizer is required")
	}
	if err := cfg.Optimizer.fitOptimizerValidate(); err != nil {
		return err
	}
	if isNilFitInterface(cfg.Loss) {
		return fmt.Errorf("fit config Loss is required")
	}
	if x == nil {
		return fmt.Errorf("fit config x is nil")
	}
	if y == nil {
		return fmt.Errorf("fit config y is nil")
	}
	if err := validateFitTargetType(cfg.Loss, y, "y"); err != nil {
		return err
	}
	if cfg.ValX == nil && cfg.ValY != nil {
		return fmt.Errorf("fit config ValX is required when ValY is provided")
	}
	if cfg.ValX != nil && cfg.ValY == nil {
		return fmt.Errorf("fit config ValY is required when ValX is provided")
	}
	if cfg.ValY != nil {
		if err := validateFitTargetType(cfg.Loss, cfg.ValY, "ValY"); err != nil {
			return err
		}
	}
	return nil
}

func validateFitTargetType(loss LossSpec, target *Tensor, name string) error {
	switch loss.(type) {
	case CrossEntropy:
		if target.dtype != DTypeInt64 {
			return fmt.Errorf("fit config Loss CrossEntropy requires %s to have int64 dtype", name)
		}
	case MSE, BCEWithLogits:
		if target.dtype != DTypeFloat32 {
			return fmt.Errorf("fit config Loss %s requires %s to have float32 dtype", loss.fitLossName(), name)
		}
	}
	return nil
}

func validateFitRows(x, y *Tensor, split string) error {
	if x == nil || y == nil {
		return fmt.Errorf("fit %s tensors must not be nil", split)
	}
	if len(x.shape) == 0 || x.shape[0] == 0 {
		return fmt.Errorf("fit %s x must have a non-empty first dimension", split)
	}
	if len(y.shape) == 0 || y.shape[0] != x.shape[0] {
		return fmt.Errorf("fit %s y first dimension %v does not match x first dimension %d", split, y.shape, x.shape[0])
	}
	return nil
}

func isNilFitInterface(value any) bool {
	if value == nil {
		return true
	}
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return rv.IsNil()
	default:
		return false
	}
}

func fitOrder(rng *rand.Rand, rows int, noShuffle bool) []int {
	if !noShuffle {
		return rng.Perm(rows)
	}
	order := make([]int, rows)
	for index := range order {
		order[index] = index
	}
	return order
}

func fitBatch(input *Tensor, rows []int) (*Tensor, error) {
	if input == nil || len(input.shape) == 0 {
		return nil, fmt.Errorf("batch input must have a leading row dimension")
	}
	shape := append([]int{len(rows)}, input.shape[1:]...)
	rowSize := input.Len() / input.shape[0]
	result, err := newTypedTensor(input.dtype, shape)
	if err != nil {
		return nil, err
	}
	for outputRow, inputRow := range rows {
		if inputRow < 0 || inputRow >= input.shape[0] {
			return nil, fmt.Errorf("batch row %d is outside input rows %d", inputRow, input.shape[0])
		}
		for offset := 0; offset < rowSize; offset++ {
			copyTensorElement(result, outputRow*rowSize+offset, input, inputRow*rowSize+offset)
		}
	}
	return result, nil
}
