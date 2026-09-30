package nn

import (
	"context"
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

// LossSpec selects the loss Fit trains with: CrossEntropy, MSE or
// BCEWithLogits, or a CustomLoss the caller computes on the tape. Its methods
// are unexported, so CustomLoss is how a caller supplies any other loss.
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

// CustomLoss selects a loss the caller computes on the tape.
//
// Loss is required. It returns the batch's loss as a float32 scalar computed
// with operations on tape, so the reverse pass reaches the parameters.
// Validate is optional: when set, Fit calls it before Loss on every batch and
// on the validation set, where the built-in losses check their targets. Name
// labels the loss in Fit's errors; empty means "CustomLoss".
type CustomLoss struct {
	Name     string
	Loss     func(tape *Tape, prediction, target *Tensor) (*Tensor, error)
	Validate func(prediction, target *Tensor) error
}

func (l CustomLoss) fitLossName() string {
	if l.Name == "" {
		return "CustomLoss"
	}
	return l.Name
}

// fitLossConfigError reports a CustomLoss that cannot run, so Fit refuses it
// before any batch.
func (l CustomLoss) fitLossConfigError() error {
	if l.Loss == nil {
		return fmt.Errorf("fit config Loss %s has no Loss function", l.fitLossName())
	}
	return nil
}

func (l CustomLoss) fitLossValidate(prediction, target *Tensor) error {
	if l.Validate == nil {
		return nil
	}
	return l.Validate(prediction, target)
}

func (l CustomLoss) fitLoss(tape *Tape, prediction, target *Tensor) (*Tensor, error) {
	if err := l.fitLossConfigError(); err != nil {
		return nil, err
	}
	loss, err := l.Loss(tape, prediction, target)
	if err != nil {
		return nil, err
	}
	name := l.fitLossName()
	if loss == nil {
		return nil, fmt.Errorf("loss %s returned no tensor", name)
	}
	if loss.dtype != DTypeFloat32 || len(loss.shape) != 0 {
		return nil, fmt.Errorf("loss %s must return a float32 scalar, got %s with shape %v", name, loss.dtype, loss.shape)
	}
	if !producedOnTape(tape, loss) {
		return nil, fmt.Errorf("loss %s returned a tensor no operation on the tape produced, so no gradient would reach the parameters", name)
	}
	return loss, nil
}

// producedOnTape reports whether an operation recorded on tape produced output.
func producedOnTape(tape *Tape, output *Tensor) bool {
	for _, op := range tape.ops {
		if op.output == output {
			return true
		}
	}
	return false
}

// SoftmaxCrossEntropy is the same loss selector as CrossEntropy.
//
// Deprecated: use CrossEntropy. Removed in the release after the one that deprecated it.
type SoftmaxCrossEntropy = CrossEntropy

// MSELoss is the same loss selector as MSE.
//
// Deprecated: use MSE. Removed in the release after the one that deprecated it.
type MSELoss = MSE

// BCEWithLogitsLoss is the same loss selector as BCEWithLogits.
//
// Deprecated: use BCEWithLogits. Removed in the release after the one that deprecated it.
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
// and optimizer methods. It is intentionally a thin, deterministic loop, and
// it is FitContext with context.Background().
func (s *Sequential) Fit(x, y *Tensor, cfg FitConfig) (*FitResult, error) {
	return s.FitContext(context.Background(), x, y, cfg)
}

// FitContext is Fit under ctx. The context is checked before every batch and
// once more after each epoch's last batch, before its validation. When it is
// done, FitContext returns ctx.Err() together with a FitResult listing only
// the epochs that finished, their validation and Progress call included. The
// model keeps every optimizer step taken before the check, including the
// steps of an epoch that did not finish. A run whose every epoch finished
// returns its result and no error, even if the context was cancelled during
// the last epoch's Progress call. A context that is already done when
// FitContext is called changes nothing. A nil context is an error.
func (s *Sequential) FitContext(ctx context.Context, x, y *Tensor, cfg FitConfig) (*FitResult, error) {
	if s == nil {
		return nil, fmt.Errorf("fit sequential is nil")
	}
	if s.tape == nil {
		return nil, fmt.Errorf("fit sequential tape is nil")
	}
	if ctx == nil {
		return nil, fmt.Errorf("fit context is nil")
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
	if err := ctx.Err(); err != nil {
		return &FitResult{}, err
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
			if err := ctx.Err(); err != nil {
				result.Elapsed = time.Since(runStarted)
				return result, err
			}
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

		if err := ctx.Err(); err != nil {
			result.Elapsed = time.Since(runStarted)
			return result, err
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
	if checked, ok := cfg.Loss.(interface{ fitLossConfigError() error }); ok {
		if err := checked.fitLossConfigError(); err != nil {
			return err
		}
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
