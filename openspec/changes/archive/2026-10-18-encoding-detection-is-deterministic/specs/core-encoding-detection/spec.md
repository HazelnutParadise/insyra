## ADDED Requirements

### Requirement: Detection gives the same answer for the same bytes
`DetectEncoding` and the reader-based detection SHALL return the same charset every time they are given the same bytes, breaking a tie between equally confident recognizers by charset name.

#### Scenario: A short Big5 file where recognizers tie
- **WHEN** the same 24-byte Big5 sample is detected 300 times
- **THEN** every call returns the same charset
