## ADDED Requirements

### Requirement: A caller's kernel cannot write outside its buffers

On Metal and Vulkan, an index past the end of a storage buffer in a caller's kernel SHALL read zero and SHALL NOT write, so no other resident buffer is changed; on Vulkan Insyra SHALL compile a caller's kernel with these bounds checks itself, and SHALL leave its own kernels and other backends unchanged.

#### Scenario: A write past the end

- **WHEN** a caller's kernel writes 64 values into a 4-value buffer on a Metal or Vulkan device
- **THEN** the 4 values are written and eight buffers allocated beside it are unchanged
