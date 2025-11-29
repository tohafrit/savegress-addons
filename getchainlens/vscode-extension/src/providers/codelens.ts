import * as vscode from 'vscode';
import { GasAnalyzer } from '../analyzers/gas';

export class CodeLensProvider implements vscode.CodeLensProvider {
    private gasAnalyzer: GasAnalyzer;
    private _onDidChangeCodeLenses: vscode.EventEmitter<void> = new vscode.EventEmitter<void>();
    public readonly onDidChangeCodeLenses: vscode.Event<void> = this._onDidChangeCodeLenses.event;

    constructor(gasAnalyzer: GasAnalyzer) {
        this.gasAnalyzer = gasAnalyzer;

        // Refresh code lenses when configuration changes
        vscode.workspace.onDidChangeConfiguration((_) => {
            this._onDidChangeCodeLenses.fire();
        });
    }

    provideCodeLenses(
        document: vscode.TextDocument,
        _token: vscode.CancellationToken
    ): vscode.ProviderResult<vscode.CodeLens[]> {
        const config = vscode.workspace.getConfiguration('getchainlens');
        if (!config.get('showGasEstimates')) {
            return [];
        }

        const codeLenses: vscode.CodeLens[] = [];
        const sourceCode = document.getText();

        // Get gas estimates for functions
        const estimates = this.gasAnalyzer.analyze(sourceCode);

        for (const estimate of estimates) {
            const line = Math.max(0, estimate.line - 1);
            const range = new vscode.Range(
                new vscode.Position(line, 0),
                new vscode.Position(line, 0)
            );

            // Gas estimate lens
            const gasIcon = this.getGasIcon(estimate.level);
            const gasLens = new vscode.CodeLens(range, {
                title: `${gasIcon} ~${estimate.estimatedGas.toLocaleString()} gas`,
                command: 'getchainlens.showGasReport',
                tooltip: `Estimated gas for ${estimate.name}. Click for detailed report.`,
            });
            codeLenses.push(gasLens);

            // Suggestions lens (if any)
            if (estimate.suggestions.length > 0) {
                const suggestionLens = new vscode.CodeLens(range, {
                    title: `💡 ${estimate.suggestions.length} optimization${estimate.suggestions.length > 1 ? 's' : ''}`,
                    command: 'getchainlens.showGasReport',
                    tooltip: estimate.suggestions.join('\n'),
                });
                codeLenses.push(suggestionLens);
            }
        }

        // Add contract-level lens
        const contractMatch = sourceCode.match(/contract\s+(\w+)/);
        if (contractMatch) {
            const contractLine = sourceCode.substring(0, sourceCode.indexOf(contractMatch[0])).split('\n').length - 1;
            const range = new vscode.Range(
                new vscode.Position(contractLine, 0),
                new vscode.Position(contractLine, 0)
            );

            const report = this.gasAnalyzer.generateReport(sourceCode);
            codeLenses.push(new vscode.CodeLens(range, {
                title: `📊 GetChainLens | ${estimates.length} functions | ~${report.totalDeploymentGas.toLocaleString()} gas deployment`,
                command: 'getchainlens.showGasReport',
                tooltip: 'Click for full gas report',
            }));
        }

        return codeLenses;
    }

    private getGasIcon(level: string): string {
        switch (level) {
            case 'low':
                return '🟢';
            case 'medium':
                return '🟡';
            case 'high':
                return '🔴';
            default:
                return '⛽';
        }
    }
}
