## ADDED Requirements

### Requirement: SafeTensors files load, validate, and refuse like ONNX does

`dl` SHALL load SafeTensors files into named tensors, validating the header
and data region at load time. A malformed file SHALL produce an error naming
the defect and the tensor involved, never a panic. Unsupported dtypes SHALL
be reported together, naming each. Values read SHALL be verified exactly
against the Python `safetensors` reference through the gated reference
harness.

#### Scenario: A PyTorch-written checkpoint loads

- **WHEN** a file written by the Python safetensors library is loaded
- **THEN** every tensor SHALL be present under its name with its declared
  shape and dtype, and every f32 value SHALL equal the reference exactly

#### Scenario: A malformed file is refused with a name

- **WHEN** a file has an oversized header length, invalid JSON, overlapping
  offsets, out-of-range regions, or an element count disagreeing with its
  shape
- **THEN** loading SHALL return an error naming the defect (and the tensor,
  where one is involved) and SHALL NOT panic

#### Scenario: Unsupported dtypes are reported together

- **WHEN** a file contains tensors of dtypes the runtime does not implement
- **THEN** the error SHALL list every offending tensor and dtype at once
