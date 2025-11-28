import * as vscode from 'vscode';
import { SecurityAnalyzer } from './analyzers/security';
import { GasAnalyzer } from './analyzers/gas';
import { DiagnosticsProvider } from './providers/diagnostics';
import { HoverProvider } from './providers/hover';
import { CodeLensProvider } from './providers/codelens';
import { ContractsTreeProvider } from './views/contracts';
import { ApiClient } from './services/api';

let diagnosticsProvider: DiagnosticsProvider;
let securityAnalyzer: SecurityAnalyzer;
let gasAnalyzer: GasAnalyzer;
let apiClient: ApiClient;

export function activate(context: vscode.ExtensionContext) {
    console.log('ChainLens is now active!');

    // Initialize analyzers
    securityAnalyzer = new SecurityAnalyzer();
    gasAnalyzer = new GasAnalyzer();

    // Initialize API client
    const config = vscode.workspace.getConfiguration('chainlens');
    apiClient = new ApiClient(config.get('apiKey') || '');

    // Initialize diagnostics
    const diagnosticCollection = vscode.languages.createDiagnosticCollection('chainlens');
    diagnosticsProvider = new DiagnosticsProvider(
        diagnosticCollection,
        securityAnalyzer,
        gasAnalyzer
    );
    context.subscriptions.push(diagnosticCollection);

    // Register hover provider
    const hoverProvider = new HoverProvider(securityAnalyzer, gasAnalyzer);
    context.subscriptions.push(
        vscode.languages.registerHoverProvider('solidity', hoverProvider)
    );

    // Register code lens provider
    const codeLensProvider = new CodeLensProvider(gasAnalyzer);
    context.subscriptions.push(
        vscode.languages.registerCodeLensProvider('solidity', codeLensProvider)
    );

    // Register tree view
    const contractsProvider = new ContractsTreeProvider();
    vscode.window.registerTreeDataProvider('chainlensContracts', contractsProvider);

    // Register commands
    context.subscriptions.push(
        vscode.commands.registerCommand('chainlens.analyzeFile', () => {
            const editor = vscode.window.activeTextEditor;
            if (editor && editor.document.languageId === 'solidity') {
                diagnosticsProvider.analyzeDocument(editor.document);
                vscode.window.showInformationMessage('ChainLens: Analysis complete');
            }
        })
    );

    context.subscriptions.push(
        vscode.commands.registerCommand('chainlens.analyzeWorkspace', async () => {
            const files = await vscode.workspace.findFiles('**/*.sol');
            let analyzed = 0;
            for (const file of files) {
                const doc = await vscode.workspace.openTextDocument(file);
                diagnosticsProvider.analyzeDocument(doc);
                analyzed++;
            }
            vscode.window.showInformationMessage(`ChainLens: Analyzed ${analyzed} contracts`);
        })
    );

    context.subscriptions.push(
        vscode.commands.registerCommand('chainlens.showGasReport', () => {
            const editor = vscode.window.activeTextEditor;
            if (editor && editor.document.languageId === 'solidity') {
                showGasReport(editor.document);
            }
        })
    );

    context.subscriptions.push(
        vscode.commands.registerCommand('chainlens.showSecurityReport', () => {
            const editor = vscode.window.activeTextEditor;
            if (editor && editor.document.languageId === 'solidity') {
                showSecurityReport(editor.document);
            }
        })
    );

    context.subscriptions.push(
        vscode.commands.registerCommand('chainlens.traceTransaction', async () => {
            const txHash = await vscode.window.showInputBox({
                prompt: 'Enter transaction hash',
                placeHolder: '0x...'
            });
            if (txHash) {
                traceTransaction(txHash);
            }
        })
    );

    // Analyze on save
    context.subscriptions.push(
        vscode.workspace.onDidSaveTextDocument((document) => {
            if (document.languageId === 'solidity' && config.get('analyzeOnSave')) {
                diagnosticsProvider.analyzeDocument(document);
            }
        })
    );

    // Analyze on open
    context.subscriptions.push(
        vscode.workspace.onDidOpenTextDocument((document) => {
            if (document.languageId === 'solidity') {
                diagnosticsProvider.analyzeDocument(document);
            }
        })
    );

    // Analyze currently open documents
    vscode.workspace.textDocuments.forEach((document) => {
        if (document.languageId === 'solidity') {
            diagnosticsProvider.analyzeDocument(document);
        }
    });
}

async function showGasReport(document: vscode.TextDocument) {
    const report = gasAnalyzer.generateReport(document.getText());

    const panel = vscode.window.createWebviewPanel(
        'chainlensGasReport',
        'ChainLens Gas Report',
        vscode.ViewColumn.Beside,
        {}
    );

    panel.webview.html = `
        <!DOCTYPE html>
        <html>
        <head>
            <style>
                body { font-family: var(--vscode-font-family); padding: 20px; }
                h1 { color: var(--vscode-editor-foreground); }
                .function { margin: 10px 0; padding: 10px; background: var(--vscode-editor-background); border-radius: 4px; }
                .function-name { font-weight: bold; color: var(--vscode-symbolIcon-functionForeground); }
                .gas { color: var(--vscode-charts-orange); }
                .low { color: var(--vscode-charts-green); }
                .medium { color: var(--vscode-charts-yellow); }
                .high { color: var(--vscode-charts-red); }
            </style>
        </head>
        <body>
            <h1>Gas Report: ${document.fileName.split('/').pop()}</h1>
            ${report.functions.map(f => `
                <div class="function">
                    <span class="function-name">${f.name}</span>
                    <span class="gas ${f.level}">${f.estimatedGas.toLocaleString()} gas</span>
                    ${f.suggestions.length > 0 ? `<ul>${f.suggestions.map(s => `<li>${s}</li>`).join('')}</ul>` : ''}
                </div>
            `).join('')}
            <p>Total estimated deployment cost: <strong>${report.totalDeploymentGas.toLocaleString()} gas</strong></p>
        </body>
        </html>
    `;
}

async function showSecurityReport(document: vscode.TextDocument) {
    const issues = securityAnalyzer.analyze(document.getText());

    const panel = vscode.window.createWebviewPanel(
        'chainlensSecurityReport',
        'ChainLens Security Report',
        vscode.ViewColumn.Beside,
        {}
    );

    const criticalCount = issues.filter(i => i.severity === 'critical').length;
    const highCount = issues.filter(i => i.severity === 'high').length;
    const mediumCount = issues.filter(i => i.severity === 'medium').length;

    panel.webview.html = `
        <!DOCTYPE html>
        <html>
        <head>
            <style>
                body { font-family: var(--vscode-font-family); padding: 20px; }
                h1 { color: var(--vscode-editor-foreground); }
                .summary { display: flex; gap: 20px; margin: 20px 0; }
                .stat { padding: 10px 20px; border-radius: 4px; }
                .critical { background: #ff000033; color: #ff6b6b; }
                .high { background: #ff880033; color: #ffa94d; }
                .medium { background: #ffff0033; color: #ffd43b; }
                .issue { margin: 10px 0; padding: 15px; background: var(--vscode-editor-background); border-radius: 4px; border-left: 3px solid; }
                .issue.critical { border-color: #ff6b6b; }
                .issue.high { border-color: #ffa94d; }
                .issue.medium { border-color: #ffd43b; }
                .issue-title { font-weight: bold; }
                .issue-location { color: var(--vscode-descriptionForeground); font-size: 12px; }
                code { background: var(--vscode-textCodeBlock-background); padding: 2px 6px; border-radius: 3px; }
            </style>
        </head>
        <body>
            <h1>Security Report: ${document.fileName.split('/').pop()}</h1>
            <div class="summary">
                <div class="stat critical">${criticalCount} Critical</div>
                <div class="stat high">${highCount} High</div>
                <div class="stat medium">${mediumCount} Medium</div>
            </div>
            ${issues.length === 0 ? '<p>No security issues found!</p>' : ''}
            ${issues.map(issue => `
                <div class="issue ${issue.severity}">
                    <div class="issue-title">${issue.type}</div>
                    <div class="issue-location">Line ${issue.line}</div>
                    <p>${issue.message}</p>
                    ${issue.suggestion ? `<p><strong>Fix:</strong> ${issue.suggestion}</p>` : ''}
                    ${issue.code ? `<code>${issue.code}</code>` : ''}
                </div>
            `).join('')}
        </body>
        </html>
    `;
}

async function traceTransaction(txHash: string) {
    const config = vscode.workspace.getConfiguration('chainlens');
    const apiKey = config.get<string>('apiKey');

    if (!apiKey) {
        const action = await vscode.window.showWarningMessage(
            'Transaction tracing requires a ChainLens API key',
            'Get API Key',
            'Enter Key'
        );
        if (action === 'Get API Key') {
            vscode.env.openExternal(vscode.Uri.parse('https://chainlens.dev/signup'));
        } else if (action === 'Enter Key') {
            vscode.commands.executeCommand('workbench.action.openSettings', 'chainlens.apiKey');
        }
        return;
    }

    vscode.window.withProgress({
        location: vscode.ProgressLocation.Notification,
        title: 'Tracing transaction...',
        cancellable: false
    }, async () => {
        try {
            const trace = await apiClient.traceTransaction(txHash);
            // Show trace in webview
            vscode.window.showInformationMessage(`Transaction traced: ${trace.status}`);
        } catch (error) {
            vscode.window.showErrorMessage(`Failed to trace transaction: ${error}`);
        }
    });
}

export function deactivate() {
    console.log('ChainLens deactivated');
}
