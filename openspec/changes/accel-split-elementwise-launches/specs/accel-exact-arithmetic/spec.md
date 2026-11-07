## ADDED Requirements

### Requirement: An elementwise resident kernel never runs as one long dispatch

The elementwise resident kernels SHALL split an array into launches of a bounded number of elements, each submitted on its own, so that no single dispatch runs long enough for an operating system to reset the GPU. Splitting SHALL NOT change any result.

#### Scenario: An array longer than a launch
- **WHEN** a fixed-point kernel, an exact-register kernel with a broadcast operand, and an in-place update each run over an array longer than two launches
- **THEN** every element holds the value computed for its own index, and each is updated exactly once

#### Scenario: A large training step on Windows
- **WHEN** a resident training step runs `tanh` and its gradient over 16.8 million elements on a GPU under Windows
- **THEN** no launch outlasts the reset and the step completes
