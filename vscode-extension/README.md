# ChainLens - Smart Contract Intelligence for VS Code

> Chrome DevTools for Blockchain Development

ChainLens provides real-time security analysis, gas estimation, and debugging capabilities for Solidity smart contracts directly in VS Code.

## Features

### 🔒 Security Analysis

Real-time detection of common vulnerabilities:

- **Reentrancy** - External calls before state changes
- **Integer Overflow** - Unchecked arithmetic (pre-0.8.0)
- **Access Control** - Missing ownership checks
- **Unchecked Returns** - Low-level call return values
- **TX Origin Auth** - Phishing vulnerability
- **Timestamp Dependency** - Miner manipulation risk
- **Delegatecall Risks** - Context manipulation
- **Selfdestruct** - Contract destruction risks

### ⛽ Gas Estimation

- Inline gas estimates for each function
- Color-coded severity (🟢 low, 🟡 medium, 🔴 high)
- Optimization suggestions
- Deployment cost estimation

### 💡 Smart Hover Information

Hover over Solidity keywords to see:
- Security implications
- Gas costs
- Best practices
- Code examples

### 📊 Reports

Generate comprehensive reports:
- Security audit report
- Gas optimization report
- Function-by-function analysis

## Installation

### From VS Code Marketplace

1. Open VS Code
2. Go to Extensions (Ctrl+Shift+X)
3. Search for "ChainLens"
4. Click Install

### From Source

```bash
cd vscode-extension
npm install
npm run compile
```

Then press F5 to launch the extension in debug mode.

## Usage

### Automatic Analysis

ChainLens automatically analyzes your Solidity files:
- On file open
- On file save
- Issues appear in the Problems panel

### Commands

- `ChainLens: Analyze Current File` - Run analysis on active file
- `ChainLens: Analyze Workspace` - Analyze all .sol files
- `ChainLens: Show Gas Report` - View detailed gas report
- `ChainLens: Show Security Report` - View security audit
- `ChainLens: Trace Transaction` - Trace a transaction (requires API key)

### Configuration

```json
{
  "chainlens.enabled": true,
  "chainlens.analyzeOnSave": true,
  "chainlens.securityLevel": "warning",
  "chainlens.showGasEstimates": true,
  "chainlens.apiKey": "",
  "chainlens.rpcUrl": ""
}
```

## Cloud Features

With a ChainLens API key, unlock additional features:

- **Transaction Tracing** - Debug any mainnet/testnet transaction
- **Fork Simulation** - Test changes against live state
- **Contract Monitoring** - Real-time alerts for your deployed contracts
- **Team Collaboration** - Share projects and findings

Get your API key at [chainlens.dev](https://chainlens.dev)

## Example

```solidity
// ChainLens will detect these issues:

function withdraw(uint256 amount) public {
    require(balances[msg.sender] >= amount);

    // ⚠️ REENTRANCY: External call before state update
    msg.sender.call{value: amount}("");

    balances[msg.sender] -= amount;
}

function onlyOwnerFunction() public {
    // ⚠️ TX_ORIGIN: Using tx.origin for auth
    require(tx.origin == owner);
}
```

## Supported Patterns

### Security Patterns Detected

| Pattern | Severity | Description |
|---------|----------|-------------|
| Reentrancy | Critical | External calls before state changes |
| Access Control | Critical | Missing ownership verification |
| Integer Overflow | High | Unchecked math (Solidity < 0.8) |
| Unchecked Call | High | Low-level call return ignored |
| TX Origin | High | Using tx.origin for auth |
| Timestamp | Medium | Block timestamp in conditions |
| Delegatecall | High | Context manipulation risk |
| Selfdestruct | High | Contract destruction |

### Gas Optimizations Suggested

- Use `external` instead of `public` for external-only functions
- Use `bytes32` instead of `string` where possible
- Use `++i` instead of `i++` in loops
- Use custom errors instead of require strings (0.8.4+)
- Use `unchecked` blocks for safe arithmetic
- Cache storage variables in memory
- Batch storage operations

## Development

### Project Structure

```
vscode-extension/
├── src/
│   ├── extension.ts        # Entry point
│   ├── analyzers/
│   │   ├── security.ts     # Security pattern detection
│   │   └── gas.ts          # Gas estimation
│   ├── providers/
│   │   ├── diagnostics.ts  # VS Code diagnostics
│   │   ├── hover.ts        # Hover information
│   │   └── codelens.ts     # Inline annotations
│   ├── services/
│   │   └── api.ts          # ChainLens API client
│   └── views/
│       └── contracts.ts    # Sidebar tree view
├── test/
│   └── fixtures/           # Test contracts
├── package.json
└── tsconfig.json
```

### Building

```bash
npm install
npm run compile
```

### Testing

```bash
npm test
```

### Packaging

```bash
npm run package
```

## Roadmap

- [ ] Vyper support
- [ ] Move language support
- [ ] Integrated debugger
- [ ] AI-powered fix suggestions
- [ ] Formal verification integration

## Contributing

Contributions are welcome! Please read our [Contributing Guide](CONTRIBUTING.md).

## License

MIT License - see [LICENSE](LICENSE)

---

Built with ❤️ by the ChainLens team
