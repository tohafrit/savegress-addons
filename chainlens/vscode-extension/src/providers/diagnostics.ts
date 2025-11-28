import * as vscode from 'vscode';
import { SecurityAnalyzer, SecurityIssue } from '../analyzers/security';
import { GasAnalyzer, GasEstimate } from '../analyzers/gas';

export class DiagnosticsProvider {
    private diagnosticCollection: vscode.DiagnosticCollection;
    private securityAnalyzer: SecurityAnalyzer;
    private gasAnalyzer: GasAnalyzer;

    constructor(
        diagnosticCollection: vscode.DiagnosticCollection,
        securityAnalyzer: SecurityAnalyzer,
        gasAnalyzer: GasAnalyzer
    ) {
        this.diagnosticCollection = diagnosticCollection;
        this.securityAnalyzer = securityAnalyzer;
        this.gasAnalyzer = gasAnalyzer;
    }

    analyzeDocument(document: vscode.TextDocument): void {
        const diagnostics: vscode.Diagnostic[] = [];
        const sourceCode = document.getText();

        // Security analysis
        const securityIssues = this.securityAnalyzer.analyze(sourceCode);
        for (const issue of securityIssues) {
            const diagnostic = this.createSecurityDiagnostic(document, issue);
            diagnostics.push(diagnostic);
        }

        // Gas analysis (only warnings for high gas functions)
        const config = vscode.workspace.getConfiguration('chainlens');
        if (config.get('showGasEstimates')) {
            const gasEstimates = this.gasAnalyzer.analyze(sourceCode);
            for (const estimate of gasEstimates) {
                if (estimate.level === 'high') {
                    const diagnostic = this.createGasDiagnostic(document, estimate);
                    diagnostics.push(diagnostic);
                }
            }
        }

        this.diagnosticCollection.set(document.uri, diagnostics);
    }

    private createSecurityDiagnostic(document: vscode.TextDocument, issue: SecurityIssue): vscode.Diagnostic {
        const line = Math.max(0, issue.line - 1);
        const lineText = document.lineAt(line).text;

        const startChar = issue.column || 0;
        const endChar = issue.endColumn || lineText.length;

        const range = new vscode.Range(
            new vscode.Position(line, startChar),
            new vscode.Position(issue.endLine ? issue.endLine - 1 : line, endChar)
        );

        const severity = this.mapSeverity(issue.severity);

        const diagnostic = new vscode.Diagnostic(
            range,
            `[ChainLens] ${issue.message}`,
            severity
        );

        diagnostic.code = issue.type;
        diagnostic.source = 'ChainLens Security';

        // Add related information with suggestion
        if (issue.suggestion) {
            diagnostic.relatedInformation = [
                new vscode.DiagnosticRelatedInformation(
                    new vscode.Location(document.uri, range),
                    `💡 ${issue.suggestion}`
                )
            ];
        }

        return diagnostic;
    }

    private createGasDiagnostic(document: vscode.TextDocument, estimate: GasEstimate): vscode.Diagnostic {
        const line = Math.max(0, estimate.line - 1);
        const lineText = document.lineAt(line).text;

        const range = new vscode.Range(
            new vscode.Position(line, 0),
            new vscode.Position(line, lineText.length)
        );

        const message = `[ChainLens] High gas function: ${estimate.name} (~${estimate.estimatedGas.toLocaleString()} gas)`;

        const diagnostic = new vscode.Diagnostic(
            range,
            message,
            vscode.DiagnosticSeverity.Warning
        );

        diagnostic.code = 'HIGH_GAS';
        diagnostic.source = 'ChainLens Gas';

        if (estimate.suggestions.length > 0) {
            diagnostic.relatedInformation = estimate.suggestions.map(suggestion =>
                new vscode.DiagnosticRelatedInformation(
                    new vscode.Location(document.uri, range),
                    `💡 ${suggestion}`
                )
            );
        }

        return diagnostic;
    }

    private mapSeverity(severity: string): vscode.DiagnosticSeverity {
        switch (severity) {
            case 'critical':
            case 'high':
                return vscode.DiagnosticSeverity.Error;
            case 'medium':
                return vscode.DiagnosticSeverity.Warning;
            case 'low':
            case 'info':
                return vscode.DiagnosticSeverity.Information;
            default:
                return vscode.DiagnosticSeverity.Hint;
        }
    }
}
