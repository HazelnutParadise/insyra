## ADDED Requirements

### Requirement: Softmax cross-entropy is a fixed sequence of correctly rounded steps
In `PrecisionFloat32Rounded`, per row with `m` the row maximum, the tape SHALL compute `d[c] = x[c] - m`, `e[c] = exp(d[c])`, `s` as the exact sum of `e` rounded once, `l = log(s)` and `r = l - d[label]`, then the loss as the exact sum of `r` over rows rounded once, divided by the row count. Any non-finite logit in a row SHALL make that row's loss NaN.

#### Scenario: Host and device agree
- **WHEN** a loss is computed on host and device from the same logits and labels
- **THEN** the loss has the same bits

### Requirement: The cross-entropy gradient is pinned
In `PrecisionFloat32Rounded`, the gradient SHALL be `g[c] = round(p[c] * scale)` with `p[c] = e[c] / s` correctly rounded and `scale = round(upstream / rows)`, and at the label `g = round(round(p*scale) - scale)`.

#### Scenario: Host and device agree
- **WHEN** the gradient is computed on host and device
- **THEN** every element has the same bits
