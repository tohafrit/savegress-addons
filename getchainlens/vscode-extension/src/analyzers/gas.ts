import * as parser from '@solidity-parser/parser';

export interface GasEstimate {
    name: string;
    estimatedGas: number;
    level: 'low' | 'medium' | 'high';
    suggestions: string[];
    line: number;
}

export interface GasReport {
    functions: GasEstimate[];
    totalDeploymentGas: number;
    suggestions: string[];
}

interface ASTNode {
    type: string;
    loc?: {
        start: { line: number; column: number };
        end: { line: number; column: number };
    };
    [key: string]: unknown;
}

// Gas costs based on EVM opcodes (approximate)
const GAS_COSTS = {
    SSTORE_NEW: 20000,      // New storage slot
    SSTORE_UPDATE: 5000,    // Update existing slot
    SLOAD: 2100,            // Read storage
    CALL: 2600,             // External call base
    LOG: 375,               // Log operation base
    LOG_DATA: 8,            // Per byte of log data
    MEMORY: 3,              // Per word of memory
    COPY: 3,                // Per word copied
    CREATE: 32000,          // Contract creation
    KECCAK256: 30,          // Hash operation base
    KECCAK256_WORD: 6,      // Per word hashed
    TRANSACTION_BASE: 21000, // Base transaction cost
    CALLDATA_ZERO: 4,       // Zero byte in calldata
    CALLDATA_NONZERO: 16,   // Non-zero byte in calldata
};

export class GasAnalyzer {
    analyze(sourceCode: string): GasEstimate[] {
        const estimates: GasEstimate[] = [];

        try {
            const ast = parser.parse(sourceCode, {
                loc: true,
                range: true,
                tolerant: true,
            });

            this.visitNode(ast as unknown as ASTNode, (node) => {
                if (node.type === 'FunctionDefinition') {
                    const estimate = this.estimateFunction(node, sourceCode);
                    if (estimate) {
                        estimates.push(estimate);
                    }
                }
            });
        } catch (error) {
            // Fallback to line-based analysis
            return this.analyzeByLines(sourceCode);
        }

        return estimates;
    }

    generateReport(sourceCode: string): GasReport {
        const functions = this.analyze(sourceCode);
        const totalDeploymentGas = this.estimateDeployment(sourceCode);
        const suggestions = this.generateOptimizationSuggestions(sourceCode);

        return {
            functions,
            totalDeploymentGas,
            suggestions,
        };
    }

    private estimateFunction(node: ASTNode, _sourceCode: string): GasEstimate | null {
        const name = node.name as string;
        if (!name) return null;

        const funcBody = node.body as ASTNode;
        if (!funcBody) {
            // Interface function or abstract
            return null;
        }

        let gas = GAS_COSTS.TRANSACTION_BASE;
        const suggestions: string[] = [];
        const statements = funcBody.statements as ASTNode[] || [];

        // Count operations
        let storageWrites = 0;
        let storageReads = 0;
        let externalCalls = 0;
        let loops = 0;
        let events = 0;

        this.countOperations(statements, {
            onStorageWrite: () => storageWrites++,
            onStorageRead: () => storageReads++,
            onExternalCall: () => externalCalls++,
            onLoop: () => loops++,
            onEvent: () => events++,
        });

        // Calculate gas
        gas += storageWrites * GAS_COSTS.SSTORE_UPDATE;
        gas += storageReads * GAS_COSTS.SLOAD;
        gas += externalCalls * GAS_COSTS.CALL;
        gas += events * GAS_COSTS.LOG;

        // Add suggestions based on analysis
        if (storageWrites > 3) {
            suggestions.push(`High storage writes (${storageWrites}). Consider batching or using memory variables.`);
        }

        if (loops > 0) {
            suggestions.push('Contains loops - gas cost may vary significantly based on iterations.');
            gas *= 1.5; // Rough estimate for loops
        }

        if (externalCalls > 2) {
            suggestions.push(`Multiple external calls (${externalCalls}). Consider multicall pattern.`);
        }

        // Check function parameters for optimization
        const params = node.parameters as ASTNode;
        if (params?.parameters) {
            const paramList = params.parameters as ASTNode[];
            for (const param of paramList) {
                const typeName = param.typeName as ASTNode;
                if ((typeName as ASTNode & {name?: string})?.name === 'string' || ((typeName as ASTNode)?.baseTypeName as ASTNode & {name?: string})?.name === 'bytes') {
                    suggestions.push('Using dynamic types (string/bytes) in parameters increases gas. Consider bytes32 if possible.');
                }
            }
        }

        // Check visibility for optimization
        const visibility = node.visibility as string;
        if (visibility === 'public') {
            suggestions.push('Consider using external instead of public for functions only called externally.');
        }

        const level = gas < 30000 ? 'low' : gas < 100000 ? 'medium' : 'high';

        return {
            name,
            estimatedGas: Math.round(gas),
            level,
            suggestions,
            line: node.loc?.start.line || 0,
        };
    }

    private countOperations(statements: ASTNode[], callbacks: {
        onStorageWrite: () => void;
        onStorageRead: () => void;
        onExternalCall: () => void;
        onLoop: () => void;
        onEvent: () => void;
    }) {
        for (const stmt of statements) {
            this.visitNode(stmt, (node) => {
                // Storage write (assignment to state variable)
                if (node.type === 'BinaryOperation') {
                    const op = node.operator as string;
                    if (['=', '+=', '-=', '*=', '/='].includes(op)) {
                        const left = node.left as ASTNode;
                        if (left?.type === 'Identifier' || left?.type === 'IndexAccess' || left?.type === 'MemberAccess') {
                            callbacks.onStorageWrite();
                        }
                    }
                }

                // External call
                if (node.type === 'FunctionCall') {
                    const expr = node.expression as ASTNode;
                    if (expr?.type === 'MemberAccess') {
                        const member = expr.memberName as string;
                        if (['call', 'send', 'transfer', 'delegatecall', 'staticcall'].includes(member)) {
                            callbacks.onExternalCall();
                        }
                    }
                }

                // Loops
                if (node.type === 'ForStatement' || node.type === 'WhileStatement' || node.type === 'DoWhileStatement') {
                    callbacks.onLoop();
                }

                // Events
                if (node.type === 'EmitStatement') {
                    callbacks.onEvent();
                }
            });
        }
    }

    private estimateDeployment(sourceCode: string): number {
        let gas = GAS_COSTS.CREATE;

        // Estimate based on code size (roughly)
        const codeSize = sourceCode.length;
        gas += codeSize * 200; // Very rough estimate

        // Count constructors and initial storage
        const storageVars = (sourceCode.match(/\b(uint|int|address|bool|bytes|string|mapping)\d*\s+\w+\s*=/g) || []).length;
        gas += storageVars * GAS_COSTS.SSTORE_NEW;

        return Math.round(gas);
    }

    private generateOptimizationSuggestions(sourceCode: string): string[] {
        const suggestions: string[] = [];

        // Check for common gas optimizations
        if (sourceCode.includes('string ')) {
            suggestions.push('Consider using bytes32 instead of string where possible (saves gas).');
        }

        if (sourceCode.match(/uint8|uint16|uint32/)) {
            suggestions.push('Smaller uint types (uint8, uint16) do not save gas in storage. Use uint256 for efficiency.');
        }

        if (sourceCode.includes('public') && !sourceCode.includes('external')) {
            suggestions.push('Functions that are only called externally should use "external" instead of "public".');
        }

        if ((sourceCode.match(/require\(/g) || []).length > 5) {
            suggestions.push('Multiple require statements. Consider using custom errors (Solidity 0.8.4+) for gas savings.');
        }

        if (sourceCode.includes('++i') === false && sourceCode.includes('i++')) {
            suggestions.push('Use ++i instead of i++ in loops for slight gas savings.');
        }

        if (!sourceCode.includes('unchecked') && sourceCode.match(/pragma\s+solidity\s+[\^~]?0\.8/)) {
            suggestions.push('Consider using "unchecked" blocks for arithmetic where overflow is impossible.');
        }

        if (sourceCode.includes('memory') && sourceCode.match(/\[\]\s+memory.*=\s*new/)) {
            suggestions.push('Pre-calculating array size saves gas vs dynamic growth.');
        }

        return suggestions;
    }

    private analyzeByLines(sourceCode: string): GasEstimate[] {
        const estimates: GasEstimate[] = [];
        const lines = sourceCode.split('\n');
        let inFunction = false;
        let funcName = '';
        let funcLine = 0;
        let gasEstimate = GAS_COSTS.TRANSACTION_BASE;
        let suggestions: string[] = [];

        lines.forEach((line, index) => {
            const funcMatch = line.match(/function\s+(\w+)/);
            if (funcMatch) {
                if (inFunction) {
                    // Save previous function
                    estimates.push({
                        name: funcName,
                        estimatedGas: Math.round(gasEstimate),
                        level: gasEstimate < 30000 ? 'low' : gasEstimate < 100000 ? 'medium' : 'high',
                        suggestions,
                        line: funcLine,
                    });
                }
                inFunction = true;
                funcName = funcMatch[1];
                funcLine = index + 1;
                gasEstimate = GAS_COSTS.TRANSACTION_BASE;
                suggestions = [];
            }

            if (inFunction) {
                // Storage operations
                if (line.match(/\w+\s*[+\-*/]?=\s*/) && !line.includes('memory') && !line.includes('//')) {
                    gasEstimate += GAS_COSTS.SSTORE_UPDATE;
                }

                // External calls
                if (line.includes('.call') || line.includes('.transfer') || line.includes('.send')) {
                    gasEstimate += GAS_COSTS.CALL;
                }

                // Events
                if (line.includes('emit ')) {
                    gasEstimate += GAS_COSTS.LOG;
                }

                // Loops
                if (line.includes('for ') || line.includes('while ')) {
                    suggestions.push('Loop detected - gas varies with iterations');
                    gasEstimate *= 1.5;
                }
            }
        });

        // Don't forget the last function
        if (inFunction) {
            estimates.push({
                name: funcName,
                estimatedGas: Math.round(gasEstimate),
                level: gasEstimate < 30000 ? 'low' : gasEstimate < 100000 ? 'medium' : 'high',
                suggestions,
                line: funcLine,
            });
        }

        return estimates;
    }

    private visitNode(node: ASTNode, callback: (node: ASTNode) => void) {
        callback(node);
        for (const key of Object.keys(node)) {
            const child = node[key];
            if (child && typeof child === 'object') {
                if (Array.isArray(child)) {
                    for (const item of child) {
                        if (item && typeof item === 'object') {
                            this.visitNode(item as ASTNode, callback);
                        }
                    }
                } else {
                    this.visitNode(child as ASTNode, callback);
                }
            }
        }
    }
}
