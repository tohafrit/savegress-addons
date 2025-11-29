import * as vscode from 'vscode';
import { SecurityAnalyzer } from './analyzers/security';
import { GasAnalyzer } from './analyzers/gas';
import { DiagnosticsProvider } from './providers/diagnostics';
import { HoverProvider } from './providers/hover';
import { CodeLensProvider } from './providers/codelens';
import { ContractsTreeProvider } from './views/contracts';
import { ApiClient, initializeApiClient, getApiClient } from './services/api';

let diagnosticsProvider: DiagnosticsProvider;
let securityAnalyzer: SecurityAnalyzer;
let gasAnalyzer: GasAnalyzer;
let apiClient: ApiClient;

export function activate(context: vscode.ExtensionContext) {
    console.log('ChainLens is now active!');

    // Initialize analyzers
    securityAnalyzer = new SecurityAnalyzer();
    gasAnalyzer = new GasAnalyzer();

    // Initialize API client with context for token persistence
    apiClient = initializeApiClient(context);

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

    // Auth commands
    context.subscriptions.push(
        vscode.commands.registerCommand('chainlens.login', async () => {
            await loginCommand();
        })
    );

    context.subscriptions.push(
        vscode.commands.registerCommand('chainlens.logout', async () => {
            apiClient.logout();
            vscode.window.showInformationMessage('ChainLens: Logged out successfully');
        })
    );

    context.subscriptions.push(
        vscode.commands.registerCommand('chainlens.showStatus', async () => {
            await showAccountStatus();
        })
    );

    context.subscriptions.push(
        vscode.commands.registerCommand('chainlens.analyzeWithBackend', async () => {
            await analyzeWithBackend();
        })
    );

    context.subscriptions.push(
        vscode.commands.registerCommand('chainlens.simulateTransaction', async () => {
            await simulateTransactionCommand();
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
    if (!apiClient.isAuthenticated()) {
        const action = await vscode.window.showWarningMessage(
            'Transaction tracing requires authentication',
            'Login',
            'Register'
        );
        if (action === 'Login') {
            vscode.commands.executeCommand('chainlens.login');
        } else if (action === 'Register') {
            vscode.env.openExternal(vscode.Uri.parse('https://chainlens.dev/signup'));
        }
        return;
    }

    const chain = await vscode.window.showQuickPick(
        ['ethereum', 'polygon', 'arbitrum'],
        { placeHolder: 'Select chain' }
    );
    if (!chain) return;

    vscode.window.withProgress({
        location: vscode.ProgressLocation.Notification,
        title: 'Tracing transaction...',
        cancellable: false
    }, async () => {
        try {
            const trace = await apiClient.traceTransaction(txHash, chain);
            showTraceResult(trace);
        } catch (error) {
            vscode.window.showErrorMessage(`Failed to trace transaction: ${error}`);
        }
    });
}

function showTraceResult(trace: any) {
    const panel = vscode.window.createWebviewPanel(
        'chainlensTrace',
        `Trace: ${trace.tx_hash.slice(0, 10)}...`,
        vscode.ViewColumn.Beside,
        {}
    );

    const statusColor = trace.status === 'success' ? '#4caf50' : '#f44336';

    panel.webview.html = `
        <!DOCTYPE html>
        <html>
        <head>
            <style>
                body { font-family: var(--vscode-font-family); padding: 20px; color: var(--vscode-editor-foreground); }
                h1 { font-size: 18px; }
                .status { color: ${statusColor}; font-weight: bold; }
                .section { margin: 15px 0; padding: 10px; background: var(--vscode-editor-background); border-radius: 4px; }
                .label { color: var(--vscode-descriptionForeground); font-size: 12px; }
                .value { font-family: monospace; word-break: break-all; }
                .call { margin: 5px 0; padding: 8px; border-left: 2px solid var(--vscode-activityBarBadge-background); margin-left: 20px; }
                .call-type { color: var(--vscode-symbolIcon-functionForeground); font-weight: bold; }
            </style>
        </head>
        <body>
            <h1>Transaction Trace</h1>
            <div class="section">
                <div class="label">Hash</div>
                <div class="value">${trace.tx_hash}</div>
            </div>
            <div class="section">
                <div class="label">Status</div>
                <div class="status">${trace.status.toUpperCase()}</div>
                ${trace.error ? `<div class="value" style="color: #f44336;">${trace.error}</div>` : ''}
            </div>
            <div class="section">
                <div class="label">From</div>
                <div class="value">${trace.from}</div>
            </div>
            <div class="section">
                <div class="label">To</div>
                <div class="value">${trace.to}</div>
            </div>
            <div class="section">
                <div class="label">Value</div>
                <div class="value">${trace.value} wei</div>
            </div>
            <div class="section">
                <div class="label">Gas Used</div>
                <div class="value">${trace.gas_used.toLocaleString()}</div>
            </div>
            <div class="section">
                <div class="label">Block</div>
                <div class="value">${trace.block_number}</div>
            </div>
            ${trace.calls && trace.calls.length > 0 ? `
                <h2>Internal Calls</h2>
                ${renderCalls(trace.calls)}
            ` : ''}
            ${trace.logs && trace.logs.length > 0 ? `
                <h2>Events (${trace.logs.length})</h2>
                ${trace.logs.map((log: any) => `
                    <div class="section">
                        <div class="label">Address</div>
                        <div class="value">${log.address}</div>
                        ${log.decoded ? `
                            <div class="label">Event</div>
                            <div class="value">${log.decoded.name}</div>
                        ` : ''}
                    </div>
                `).join('')}
            ` : ''}
        </body>
        </html>
    `;
}

function renderCalls(calls: any[], depth = 0): string {
    return calls.map(call => `
        <div class="call" style="margin-left: ${depth * 20}px;">
            <span class="call-type">${call.type}</span>
            <span class="value">${call.to}</span>
            ${call.error ? `<div style="color: #f44336;">Error: ${call.error}</div>` : ''}
            ${call.gas_used ? `<div class="label">Gas: ${call.gas_used.toLocaleString()}</div>` : ''}
            ${call.calls ? renderCalls(call.calls, depth + 1) : ''}
        </div>
    `).join('');
}

async function loginCommand() {
    const email = await vscode.window.showInputBox({
        prompt: 'Enter your email',
        placeHolder: 'email@example.com'
    });
    if (!email) return;

    const password = await vscode.window.showInputBox({
        prompt: 'Enter your password',
        password: true
    });
    if (!password) return;

    try {
        const result = await apiClient.login(email, password);
        vscode.window.showInformationMessage(`ChainLens: Welcome, ${result.user.name}!`);
    } catch (error: any) {
        const message = error.response?.data?.error || error.message;
        vscode.window.showErrorMessage(`Login failed: ${message}`);
    }
}

async function showAccountStatus() {
    if (!apiClient.isAuthenticated()) {
        vscode.window.showWarningMessage('ChainLens: Not logged in');
        return;
    }

    try {
        const user = await apiClient.getCurrentUser();
        const usage = await apiClient.getUsage();

        const panel = vscode.window.createWebviewPanel(
            'chainlensAccount',
            'ChainLens Account',
            vscode.ViewColumn.One,
            {}
        );

        const planColor = user.plan === 'enterprise' ? '#ffd700' : user.plan === 'pro' ? '#4caf50' : '#90a4ae';

        panel.webview.html = `
            <!DOCTYPE html>
            <html>
            <head>
                <style>
                    body { font-family: var(--vscode-font-family); padding: 20px; }
                    h1 { color: var(--vscode-editor-foreground); }
                    .section { margin: 20px 0; padding: 15px; background: var(--vscode-editor-background); border-radius: 8px; }
                    .label { color: var(--vscode-descriptionForeground); font-size: 12px; margin-bottom: 5px; }
                    .value { font-size: 16px; color: var(--vscode-editor-foreground); }
                    .plan { color: ${planColor}; font-weight: bold; text-transform: uppercase; }
                    .usage-bar { height: 8px; background: var(--vscode-progressBar-background); border-radius: 4px; margin-top: 10px; }
                    .usage-fill { height: 100%; background: var(--vscode-progressBar-foreground); border-radius: 4px; }
                </style>
            </head>
            <body>
                <h1>Account</h1>
                <div class="section">
                    <div class="label">Name</div>
                    <div class="value">${user.name}</div>
                </div>
                <div class="section">
                    <div class="label">Email</div>
                    <div class="value">${user.email}</div>
                </div>
                <div class="section">
                    <div class="label">Plan</div>
                    <div class="value plan">${user.plan}</div>
                </div>
                <div class="section">
                    <div class="label">API Usage</div>
                    <div class="value">${usage.api_calls_used.toLocaleString()} / ${usage.api_calls_limit.toLocaleString()}</div>
                    <div class="usage-bar">
                        <div class="usage-fill" style="width: ${Math.min(100, (usage.api_calls_used / usage.api_calls_limit) * 100)}%;"></div>
                    </div>
                    <div class="label" style="margin-top: 5px;">Resets: ${usage.period_end}</div>
                </div>
            </body>
            </html>
        `;
    } catch (error) {
        vscode.window.showErrorMessage(`Failed to get account info: ${error}`);
    }
}

async function analyzeWithBackend() {
    const editor = vscode.window.activeTextEditor;
    if (!editor || editor.document.languageId !== 'solidity') {
        vscode.window.showWarningMessage('Please open a Solidity file');
        return;
    }

    if (!apiClient.isAuthenticated()) {
        vscode.window.showWarningMessage('Please login first');
        return;
    }

    const source = editor.document.getText();

    vscode.window.withProgress({
        location: vscode.ProgressLocation.Notification,
        title: 'Analyzing contract with ChainLens...',
        cancellable: false
    }, async () => {
        try {
            const result = await apiClient.analyzeContract(source);

            // Update diagnostics from backend results
            const diagnostics: vscode.Diagnostic[] = result.issues.map(issue => {
                const range = new vscode.Range(
                    issue.line - 1, 0,
                    issue.line - 1, 100
                );
                const severity = issue.severity === 'critical' || issue.severity === 'high'
                    ? vscode.DiagnosticSeverity.Error
                    : issue.severity === 'medium'
                        ? vscode.DiagnosticSeverity.Warning
                        : vscode.DiagnosticSeverity.Information;

                const diagnostic = new vscode.Diagnostic(range, issue.message, severity);
                diagnostic.source = 'ChainLens';
                diagnostic.code = issue.id;
                return diagnostic;
            });

            const collection = vscode.languages.createDiagnosticCollection('chainlens-backend');
            collection.set(editor.document.uri, diagnostics);

            // Show summary
            vscode.window.showInformationMessage(
                `Analysis complete: Security ${result.score.security}/100, Gas ${result.score.gas_efficiency}/100, Quality ${result.score.code_quality}/100`
            );
        } catch (error) {
            vscode.window.showErrorMessage(`Analysis failed: ${error}`);
        }
    });
}

async function simulateTransactionCommand() {
    if (!apiClient.isAuthenticated()) {
        vscode.window.showWarningMessage('Please login first');
        return;
    }

    const chain = await vscode.window.showQuickPick(
        ['ethereum', 'polygon', 'arbitrum'],
        { placeHolder: 'Select chain' }
    );
    if (!chain) return;

    const to = await vscode.window.showInputBox({
        prompt: 'Contract address',
        placeHolder: '0x...'
    });
    if (!to) return;

    const data = await vscode.window.showInputBox({
        prompt: 'Calldata (hex)',
        placeHolder: '0x...'
    });
    if (!data) return;

    const from = await vscode.window.showInputBox({
        prompt: 'From address (optional)',
        placeHolder: '0x...'
    });

    vscode.window.withProgress({
        location: vscode.ProgressLocation.Notification,
        title: 'Simulating transaction...',
        cancellable: false
    }, async () => {
        try {
            const result = await apiClient.simulateTransaction(
                chain,
                from || '0x0000000000000000000000000000000000000000',
                to,
                data
            );

            if (result.success) {
                vscode.window.showInformationMessage(
                    `Simulation succeeded! Gas used: ${result.gas_used.toLocaleString()}`
                );
            } else {
                vscode.window.showWarningMessage(
                    `Simulation reverted: ${result.revert?.message || result.error || 'Unknown error'}`
                );
            }
        } catch (error) {
            vscode.window.showErrorMessage(`Simulation failed: ${error}`);
        }
    });
}

export function deactivate() {
    console.log('ChainLens deactivated');
}
