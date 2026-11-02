package nn

// ClearOptions selects which gradients survive Tape.Clear.
type ClearOptions struct {
	// KeepGradients keeps only parameter gradients, usable by Grad and the
	// built-in optimizers. The next successful Backward replaces them.
	KeepGradients bool
}

// SetClearOptions configures every subsequent Clear. The zero options discard
// all gradients. Close always releases all gradients, regardless of this policy.
func (t *Tape) SetClearOptions(options ClearOptions) { t.clearOptions = options }

// GetClearOptions returns the current Clear policy.
func (t *Tape) GetClearOptions() ClearOptions { return t.clearOptions }

// SetFrozen controls whether SGD, SGDMomentum, Adam and AdamW skip this
// parameter. Its optimizer state stays for unfreezing. Gradients are still
// computed, and explicit caller-managed Update calls remain available.
func (p *Parameter) SetFrozen(frozen bool) {
	if p != nil {
		p.frozen = frozen
	}
}

// GetFrozen reports whether built-in optimizers skip this parameter.
func (p *Parameter) GetFrozen() bool { return p != nil && p.frozen }

func (t *Tape) clearStep(keepGradients bool) {
	t.ops = nil
	grads := make(map[*Tensor]*Tensor)
	kept := make(map[*Tensor]bool)
	for _, parameter := range t.params {
		var gradient *Tensor
		if keepGradients {
			gradient = t.grads[parameter.value]
		}
		parameter.grad = gradient
		if gradient != nil {
			grads[parameter.value] = gradient
			kept[gradient] = true
		}
	}
	// A retained gradient may also be an operation output, and several
	// parameters may share it. Keep one owner across both ownership lists.
	var owned []*Tensor
	seen := make(map[*Tensor]bool)
	for _, tensors := range [][]*Tensor{t.owned, t.gradOwned} {
		for _, tensor := range tensors {
			if seen[tensor] || t.marked[tensor] != nil {
				continue
			}
			seen[tensor] = true
			if kept[tensor] {
				owned = append(owned, tensor)
			} else {
				tensor.Release()
			}
		}
	}
	for _, release := range t.releasers {
		release()
	}
	t.releasers, t.owned = nil, nil
	t.gradOwned, t.grads = owned, grads
}

// releaseGradients drops only ownership that no live graph, parameter or
// successful reverse pass still needs. Custom rules can return an existing
// tensor, and a retained gradient can be marked as a parameter between passes.
func (t *Tape) releaseGradients(tensors []*Tensor, keep ...[]*Tensor) {
	if len(tensors) == 0 {
		return
	}
	protected := make(map[*Tensor]bool)
	for _, group := range keep {
		for _, tensor := range group {
			protected[tensor] = true
		}
	}
	for _, tensor := range tensors {
		if !protected[tensor] && t.marked[tensor] == nil {
			tensor.Release()
			protected[tensor] = true
		}
	}
}
