## ADDED Requirements

### Requirement: Adam moments are two-term sums rounded once
In `PrecisionFloat32Rounded`, `Adam` and `AdamW` SHALL compute `m` as the exact sum `b1*m + c1*g` rounded once, `g2` as `g*g` rounded, and `v` as the exact sum `b2*v + c2*g2` rounded once, where `c1` and `c2` are `1-b1` and `1-b2` rounded once to float32 on the host. The result SHALL be the same bits on the host and on every device.

#### Scenario: Host and device agree
- **WHEN** one Adam step runs on the host and on a device from the same state
- **THEN** the moments have the same bits

### Requirement: Adam updates divide and take square roots in float32
In `PrecisionFloat32Rounded`, `Adam` SHALL compute `mhat = m / bias1`, `vhat = v / bias2`, `den = round(sqrt(vhat) + eps)` and `w = round(w + (-lr) * round(mhat / den))`, each division and square root correctly rounded. `bias1`, `bias2` and `eps` SHALL be float32 scalars computed once on the host. A non-finite gradient SHALL propagate to the parameter as NaN or infinity as IEEE arithmetic gives.

#### Scenario: A parameter update
- **WHEN** an Adam step runs on host and device with the same state and gradient
- **THEN** the updated parameter has the same bits

### Requirement: AdamW decays the parameter before the moments
In `PrecisionFloat32Rounded`, `AdamW` SHALL first replace `w` with the exact sum `w + (-lrwd) * w` rounded once, where `lrwd` is `lr*wd` rounded once to float32 on the host, and then apply the Adam update to that value.

#### Scenario: Decay on both sides
- **WHEN** an AdamW step runs on host and device
- **THEN** the decayed and updated parameters have the same bits

### Requirement: The default tape keeps its AdamW
A tape that is not in `PrecisionFloat32Rounded` SHALL compute `Adam` and `AdamW` with the float64 intermediates of earlier releases.

#### Scenario: No precision set
- **WHEN** a tape never calls `SetPrecision`
- **THEN** `AdamW` returns exactly the values of the previous release
