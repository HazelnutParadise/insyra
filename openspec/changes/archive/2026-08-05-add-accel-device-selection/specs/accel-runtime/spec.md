# accel-runtime Delta: add-accel-device-selection

## ADDED Requirements

### Requirement: Callers can bound which devices a session may use
The system SHALL honor a process-wide hard device mask from `INSYRA_ACCEL_DEVICES` applied at the discovery boundary and a per-session hard allowlist from `Config.Devices`, SHALL treat the intersection as the eligible set, SHALL keep soft preference ordering within that set, and SHALL leave behavior unchanged when neither bound is given.

#### Scenario: The environment mask hides a device

- **WHEN** `INSYRA_ACCEL_DEVICES` names a subset of discovered devices
- **THEN** devices outside the subset are invisible to every session and every downstream consumer

#### Scenario: The config allowlist pins a session

- **WHEN** `Config.Devices` names devices
- **THEN** the session uses only devices inside both the allowlist and the environment mask

#### Scenario: Strict mode with nothing eligible errors

- **WHEN** the bounds leave the eligible set empty under a strict mode
- **THEN** the session or operation returns an error naming the bound that emptied the set

#### Scenario: Automatic mode with nothing eligible falls back observably

- **WHEN** the bounds leave the eligible set empty under an automatic mode
- **THEN** the operation completes on the CPU and reports a fallback reason naming the device bounds

#### Scenario: An entry matching no device is surfaced

- **WHEN** a bound names a device that does not exist
- **THEN** the mismatch is visible in the session's reporting rather than silently ignored
