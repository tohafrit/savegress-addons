import * as vscode from 'vscode';
import { SecurityAnalyzer } from '../analyzers/security';
import { GasAnalyzer } from '../analyzers/gas';

export class HoverProvider implements vscode.HoverProvider {
    private securityAnalyzer: SecurityAnalyzer;
    private gasAnalyzer: GasAnalyzer;

    constructor(securityAnalyzer: SecurityAnalyzer, gasAnalyzer: GasAnalyzer) {
        this.securityAnalyzer = securityAnalyzer;
        this.gasAnalyzer = gasAnalyzer;
    }

    provideHover(
        document: vscode.TextDocument,
        position: vscode.Position,
        _token: vscode.CancellationToken
    ): vscode.ProviderResult<vscode.Hover> {
        const line = document.lineAt(position.line);
        const lineText = line.text;
        const wordRange = document.getWordRangeAtPosition(position);
        const word = wordRange ? document.getText(wordRange) : '';

        const contents: vscode.MarkdownString[] = [];

        // Check for security-related patterns
        const securityInfo = this.getSecurityInfo(lineText, word);
        if (securityInfo) {
            contents.push(securityInfo);
        }

        // Check for gas-related patterns
        const gasInfo = this.getGasInfo(lineText, word);
        if (gasInfo) {
            contents.push(gasInfo);
        }

        // Check for Solidity built-ins
        const builtinInfo = this.getBuiltinInfo(word);
        if (builtinInfo) {
            contents.push(builtinInfo);
        }

        if (contents.length === 0) {
            return null;
        }

        return new vscode.Hover(contents, wordRange);
    }

    private getSecurityInfo(lineText: string, word: string): vscode.MarkdownString | null {
        const md = new vscode.MarkdownString();
        md.isTrusted = true;

        // tx.origin
        if (word === 'origin' && lineText.includes('tx.origin')) {
            md.appendMarkdown('### ⚠️ ChainLens Security Warning\n\n');
            md.appendMarkdown('**`tx.origin`** returns the original sender of the transaction.\n\n');
            md.appendMarkdown('**Risk:** Using `tx.origin` for authorization is vulnerable to phishing attacks.\n\n');
            md.appendMarkdown('**Recommendation:** Use `msg.sender` instead.\n\n');
            md.appendCodeblock('// Bad\nrequire(tx.origin == owner);\n\n// Good\nrequire(msg.sender == owner);', 'solidity');
            return md;
        }

        // selfdestruct
        if (word === 'selfdestruct') {
            md.appendMarkdown('### ⚠️ ChainLens Security Warning\n\n');
            md.appendMarkdown('**`selfdestruct`** destroys the contract and sends remaining ETH to target.\n\n');
            md.appendMarkdown('**Risks:**\n');
            md.appendMarkdown('- Contract becomes unusable after destruction\n');
            md.appendMarkdown('- Can be used maliciously if access control is missing\n');
            md.appendMarkdown('- May be deprecated in future Ethereum upgrades\n\n');
            md.appendMarkdown('**Recommendation:** Add strict access control and consider alternatives.');
            return md;
        }

        // delegatecall
        if (word === 'delegatecall') {
            md.appendMarkdown('### ⚠️ ChainLens Security Warning\n\n');
            md.appendMarkdown('**`delegatecall`** executes code in the context of the calling contract.\n\n');
            md.appendMarkdown('**Risks:**\n');
            md.appendMarkdown('- Can modify caller\'s storage\n');
            md.appendMarkdown('- Storage layout must match between contracts\n');
            md.appendMarkdown('- Dangerous if target address is user-controlled\n\n');
            md.appendMarkdown('**Recommendation:** Only delegatecall to trusted, audited contracts.');
            return md;
        }

        // call with value
        if (lineText.includes('.call{value')) {
            md.appendMarkdown('### 💡 ChainLens Tip\n\n');
            md.appendMarkdown('Low-level `.call{value: ...}()` is the recommended way to send ETH.\n\n');
            md.appendMarkdown('**Important:** Always check the return value!\n\n');
            md.appendCodeblock('(bool success, ) = recipient.call{value: amount}("");\nrequire(success, "Transfer failed");', 'solidity');
            return md;
        }

        // reentrancy pattern
        if (lineText.includes('.call') || lineText.includes('.transfer') || lineText.includes('.send')) {
            if (!lineText.trim().startsWith('//')) {
                md.appendMarkdown('### 💡 ChainLens Tip: Reentrancy Prevention\n\n');
                md.appendMarkdown('External calls can trigger reentrancy attacks.\n\n');
                md.appendMarkdown('**Follow Checks-Effects-Interactions pattern:**\n');
                md.appendCodeblock('// 1. Checks\nrequire(balance >= amount);\n\n// 2. Effects (update state FIRST)\nbalance -= amount;\n\n// 3. Interactions (external call LAST)\n(bool success, ) = msg.sender.call{value: amount}("");', 'solidity');
                return md;
            }
        }

        return null;
    }

    private getGasInfo(lineText: string, word: string): vscode.MarkdownString | null {
        const md = new vscode.MarkdownString();
        md.isTrusted = true;

        // Storage operations
        if (lineText.match(/\w+\s*=\s*/) && !lineText.includes('memory') && !lineText.includes('//')) {
            if (lineText.includes('mapping') || lineText.includes('[')) {
                md.appendMarkdown('### ⛽ ChainLens Gas Info\n\n');
                md.appendMarkdown('**Storage write:** ~5,000-20,000 gas\n\n');
                md.appendMarkdown('- New slot: 20,000 gas\n');
                md.appendMarkdown('- Update existing: 5,000 gas\n');
                md.appendMarkdown('- Zero → non-zero: 20,000 gas\n');
                md.appendMarkdown('- Non-zero → zero: Refund ~15,000 gas\n');
                return md;
            }
        }

        // keccak256
        if (word === 'keccak256') {
            md.appendMarkdown('### ⛽ ChainLens Gas Info\n\n');
            md.appendMarkdown('**`keccak256`** hash function costs:\n');
            md.appendMarkdown('- Base cost: 30 gas\n');
            md.appendMarkdown('- Per 32-byte word: 6 gas\n\n');
            md.appendMarkdown('💡 Consider caching hash results if used multiple times.');
            return md;
        }

        // array operations
        if (word === 'push') {
            md.appendMarkdown('### ⛽ ChainLens Gas Info\n\n');
            md.appendMarkdown('**Array `push`** writes to storage:\n');
            md.appendMarkdown('- New element: ~20,000 gas\n');
            md.appendMarkdown('- Updates array length: ~5,000 gas\n\n');
            md.appendMarkdown('💡 Consider batching operations or using fixed-size arrays.');
            return md;
        }

        return null;
    }

    private getBuiltinInfo(word: string): vscode.MarkdownString | null {
        const md = new vscode.MarkdownString();
        md.isTrusted = true;

        const builtins: Record<string, string> = {
            'msg': '### `msg` Global\n\n- `msg.sender` - Address of immediate caller\n- `msg.value` - ETH sent with call (in wei)\n- `msg.data` - Complete calldata\n- `msg.sig` - First 4 bytes (function selector)',
            'block': '### `block` Global\n\n- `block.timestamp` - Current block timestamp\n- `block.number` - Current block number\n- `block.basefee` - Base fee (EIP-1559)\n- `block.chainid` - Current chain ID\n- `block.coinbase` - Miner address',
            'require': '### `require(condition, message)`\n\nReverts if condition is false.\n\n⛽ Gas: Refunds remaining gas on failure.\n\n💡 In Solidity 0.8.4+, consider custom errors for gas savings.',
            'revert': '### `revert(message)`\n\nReverts the transaction with an error message.\n\n⛽ Gas: Refunds remaining gas.\n\n💡 Use custom errors for cheaper reverts.',
            'assert': '### `assert(condition)`\n\nUsed for invariant checking. Consumes all gas on failure.\n\n⚠️ Only use for conditions that should NEVER be false.',
            'payable': '### `payable` Modifier\n\nAllows a function or address to receive ETH.\n\n```solidity\nfunction deposit() public payable {\n    // msg.value contains sent ETH\n}\n```',
            'view': '### `view` Modifier\n\nFunction that reads but doesn\'t modify state.\n\n⛽ Gas: Free when called externally (not in transaction).',
            'pure': '### `pure` Modifier\n\nFunction that neither reads nor modifies state.\n\n⛽ Gas: Free when called externally.',
            'external': '### `external` Visibility\n\nFunction can only be called from outside the contract.\n\n⛽ Gas: More efficient than `public` for external calls.',
            'internal': '### `internal` Visibility\n\nFunction can only be called from this contract or derived contracts.\n\nSimilar to `protected` in other languages.',
            'private': '### `private` Visibility\n\nFunction can only be called from this contract.\n\n⚠️ Note: Private data is still visible on-chain!',
            'memory': '### `memory` Location\n\nTemporary storage that exists during function execution.\n\n⛽ Gas: Much cheaper than storage (~3 gas per word).',
            'storage': '### `storage` Location\n\nPersistent data stored on-chain.\n\n⛽ Gas: Expensive! 20,000 gas for new slots, 5,000 for updates.',
            'calldata': '### `calldata` Location\n\nRead-only location for function parameters.\n\n⛽ Gas: Most efficient for external function parameters.',
        };

        const info = builtins[word];
        if (info) {
            md.appendMarkdown(info);
            return md;
        }

        return null;
    }
}
