# Changelog

All notable changes to the ChainLens VS Code extension will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2024-11-28

### Added

- **Security Analysis** - Real-time detection of common smart contract vulnerabilities:
  - Reentrancy attacks (Critical)
  - Unchecked external calls (High)
  - tx.origin authentication (High)
  - Integer overflow/underflow for Solidity < 0.8.0 (High)
  - Timestamp dependency (Medium)
  - Delegatecall usage (High)
  - Selfdestruct usage (High)
  - Missing access control (Medium)
  - Unprotected withdrawal functions (Critical)
  - Inline assembly usage (Info)

- **Gas Estimation** - Function-level gas cost analysis with optimization suggestions:
  - EVM operation cost breakdown
  - Storage vs memory usage analysis
  - Optimization recommendations (string→bytes32, public→external, etc.)
  - Deployment cost estimation

- **VS Code Integration**:
  - Inline diagnostics with severity levels
  - Hover information for security issues and gas costs
  - CodeLens for quick actions
  - Command palette integration
  - Automatic analysis on file open/save
  - Webview reports for security and gas analysis

- **Configuration Options**:
  - Enable/disable analysis
  - Set minimum severity level
  - Toggle gas estimates display
  - API key for cloud features (coming soon)
  - Custom RPC URL for transaction tracing (coming soon)

### Security Detectors

| Detector | Severity | Description |
|----------|----------|-------------|
| Reentrancy | Critical | Detects potential reentrancy vulnerabilities |
| Unchecked Call | High | External calls without return value checks |
| tx.origin Auth | High | Using tx.origin for authentication |
| Integer Overflow | High | Overflow risks in Solidity < 0.8.0 |
| Timestamp Dependency | Medium | Block timestamp usage in critical logic |
| Delegatecall | High | Delegatecall to potentially unsafe targets |
| Selfdestruct | High | Contract can be destroyed |
| Access Control | Medium | Public functions modifying state |
| Unprotected Withdrawal | Critical | Withdrawal without access control |
| Inline Assembly | Info | Usage of inline assembly |

## [Unreleased]

### Planned

- Transaction tracing integration
- Fork simulation
- Contract monitoring
- Cloud-based analysis API
- Multi-file project analysis
- Foundry/Hardhat integration
