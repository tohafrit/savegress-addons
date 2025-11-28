import * as vscode from 'vscode';
import * as path from 'path';

export class ContractsTreeProvider implements vscode.TreeDataProvider<ContractItem> {
    private _onDidChangeTreeData: vscode.EventEmitter<ContractItem | undefined | null | void> = new vscode.EventEmitter<ContractItem | undefined | null | void>();
    readonly onDidChangeTreeData: vscode.Event<ContractItem | undefined | null | void> = this._onDidChangeTreeData.event;

    constructor() {
        // Refresh when files change
        vscode.workspace.onDidChangeTextDocument(() => this.refresh());
        vscode.workspace.onDidCreateFiles(() => this.refresh());
        vscode.workspace.onDidDeleteFiles(() => this.refresh());
    }

    refresh(): void {
        this._onDidChangeTreeData.fire();
    }

    getTreeItem(element: ContractItem): vscode.TreeItem {
        return element;
    }

    async getChildren(element?: ContractItem): Promise<ContractItem[]> {
        if (!vscode.workspace.workspaceFolders) {
            return [];
        }

        if (!element) {
            // Root level - show contract files
            return this.getContractFiles();
        }

        // Child level - show contracts within file
        return this.getContractsInFile(element.resourceUri!);
    }

    private async getContractFiles(): Promise<ContractItem[]> {
        const files = await vscode.workspace.findFiles('**/*.sol', '**/node_modules/**');

        return files.map(file => {
            const fileName = path.basename(file.fsPath);
            return new ContractItem(
                fileName,
                vscode.TreeItemCollapsibleState.Collapsed,
                file,
                'file'
            );
        }).sort((a, b) => a.label!.toString().localeCompare(b.label!.toString()));
    }

    private async getContractsInFile(fileUri: vscode.Uri): Promise<ContractItem[]> {
        const document = await vscode.workspace.openTextDocument(fileUri);
        const text = document.getText();
        const items: ContractItem[] = [];

        // Find contracts
        const contractRegex = /\b(contract|interface|library|abstract\s+contract)\s+(\w+)/g;
        let match;

        while ((match = contractRegex.exec(text)) !== null) {
            const type = match[1].replace('abstract ', '');
            const name = match[2];
            const line = text.substring(0, match.index).split('\n').length - 1;

            const item = new ContractItem(
                name,
                vscode.TreeItemCollapsibleState.None,
                fileUri,
                type as 'contract' | 'interface' | 'library',
                line
            );

            // Add command to go to definition
            item.command = {
                command: 'vscode.open',
                title: 'Open Contract',
                arguments: [
                    fileUri,
                    { selection: new vscode.Range(line, 0, line, 0) }
                ]
            };

            items.push(item);
        }

        return items;
    }
}

class ContractItem extends vscode.TreeItem {
    constructor(
        public readonly label: string,
        public readonly collapsibleState: vscode.TreeItemCollapsibleState,
        public readonly resourceUri?: vscode.Uri,
        public readonly itemType?: 'file' | 'contract' | 'interface' | 'library',
        public readonly line?: number
    ) {
        super(label, collapsibleState);

        this.tooltip = this.getTooltip();
        this.iconPath = this.getIcon();
        this.contextValue = itemType;

        if (itemType === 'file' && resourceUri) {
            this.description = this.getFileDescription(resourceUri);
        }
    }

    private getTooltip(): string {
        switch (this.itemType) {
            case 'contract':
                return `Contract: ${this.label}`;
            case 'interface':
                return `Interface: ${this.label}`;
            case 'library':
                return `Library: ${this.label}`;
            case 'file':
                return this.resourceUri?.fsPath || '';
            default:
                return this.label;
        }
    }

    private getIcon(): vscode.ThemeIcon {
        switch (this.itemType) {
            case 'contract':
                return new vscode.ThemeIcon('symbol-class', new vscode.ThemeColor('symbolIcon.classForeground'));
            case 'interface':
                return new vscode.ThemeIcon('symbol-interface', new vscode.ThemeColor('symbolIcon.interfaceForeground'));
            case 'library':
                return new vscode.ThemeIcon('library', new vscode.ThemeColor('symbolIcon.moduleForeground'));
            case 'file':
                return new vscode.ThemeIcon('file-code');
            default:
                return new vscode.ThemeIcon('symbol-misc');
        }
    }

    private getFileDescription(uri: vscode.Uri): string {
        const workspaceFolder = vscode.workspace.getWorkspaceFolder(uri);
        if (workspaceFolder) {
            return path.relative(workspaceFolder.uri.fsPath, path.dirname(uri.fsPath));
        }
        return '';
    }
}
