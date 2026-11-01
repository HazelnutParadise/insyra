## ADDED Requirements

### Requirement: Built-in kernels compile in the background

The first resident buffer made on a device SHALL start compiling every built-in kernel in the background without holding the submission lock, so a computation waits for at most the kernel it needs; a compiled kernel SHALL be installed only once, and one a computation built first SHALL be released.

#### Scenario: The first buffer on a device

- **WHEN** a resident buffer is made on a device and the background compile finishes
- **THEN** every built-in kernel is built and none reports an error

### Requirement: Background compiling survives a lost device

A kernel compiled in the background but not installed when its device is lost SHALL be released before the device is released, and a background compile whose device was released SHALL stop without compiling.

#### Scenario: A loss between compile and install

- **WHEN** a lost device is released while a compiled kernel waits to be installed
- **THEN** that kernel is released first and the waiting slot is empty
