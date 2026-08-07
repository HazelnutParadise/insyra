## ADDED Requirements

### Requirement: A CNN trains one Adam step matching PyTorch

The tape SHALL differentiate the CNN family — Conv with input, weight, and
bias gradients under the exercised attribute combinations, MaxPool routing
to the selected maxima, AveragePool and GlobalAveragePool spreading under
the forward's count_include_pad semantics, and inference-mode
BatchNormalization with input, scale, and bias gradients over loaded
statistics. A fixed MNIST-class CNN loaded from one SafeTensors file by
both PyTorch (eval-mode BatchNorm) and dl SHALL agree on the loss, every
parameter gradient, and every post-step parameter within f32 tolerance
after one Adam step.

#### Scenario: One CNN Adam step agrees end to end

- **WHEN** the same CNN weights are loaded by PyTorch and dl and both run
  one forward, backward, and Adam step on the same fixed batch
- **THEN** loss, every gradient, and every updated parameter SHALL agree
  within f32 tolerance through the reference gate

#### Scenario: Every CNN VJP survives finite differences without a toolchain

- **WHEN** each CNN-family VJP is compared against central finite
  differences on tiny shapes, including grouped Conv and asymmetric pads
- **THEN** they SHALL agree within the tolerance the test derives from the
  perturbation

#### Scenario: Training-mode batch statistics are refused

- **WHEN** backpropagation would require gradients of batch-computed mean
  or variance
- **THEN** the tape SHALL refuse naming the operation, matching the
  forward's inference-only contract
