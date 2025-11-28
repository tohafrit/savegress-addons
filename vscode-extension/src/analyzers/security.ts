import * as parser from '@solidity-parser/parser';

export interface SecurityIssue {
    type: string;
    severity: 'critical' | 'high' | 'medium' | 'low' | 'info';
    line: number;
    column: number;
    endLine?: number;
    endColumn?: number;
    message: string;
    suggestion?: string;
    code?: string;
}

interface ASTNode {
    type: string;
    loc?: {
        start: { line: number; column: number };
        end: { line: number; column: number };
    };
    [key: string]: unknown;
}

export class SecurityAnalyzer {
    private patterns: SecurityPattern[] = [
        new ReentrancyPattern(),
        new UncheckedCallPattern(),
        new TxOriginPattern(),
        new IntegerOverflowPattern(),
        new TimestampDependencyPattern(),
        new DelegateCallPattern(),
        new SelfDestructPattern(),
        new ArbitraryJumpPattern(),
        new UnprotectedEtherWithdrawalPattern(),
        new AccessControlPattern(),
    ];

    analyze(sourceCode: string): SecurityIssue[] {
        const issues: SecurityIssue[] = [];

        try {
            const ast = parser.parse(sourceCode, {
                loc: true,
                range: true,
                tolerant: true,
            });

            for (const pattern of this.patterns) {
                const patternIssues = pattern.check(ast, sourceCode);
                issues.push(...patternIssues);
            }
        } catch (error) {
            // Parse error - still try line-based analysis
            const lineIssues = this.analyzeByLines(sourceCode);
            issues.push(...lineIssues);
        }

        return issues.sort((a, b) => {
            const severityOrder = { critical: 0, high: 1, medium: 2, low: 3, info: 4 };
            return severityOrder[a.severity] - severityOrder[b.severity];
        });
    }

    private analyzeByLines(sourceCode: string): SecurityIssue[] {
        const issues: SecurityIssue[] = [];
        const lines = sourceCode.split('\n');

        lines.forEach((line, index) => {
            const lineNum = index + 1;

            // tx.origin check
            if (line.includes('tx.origin') && !line.trim().startsWith('//')) {
                issues.push({
                    type: 'TX_ORIGIN_AUTH',
                    severity: 'high',
                    line: lineNum,
                    column: line.indexOf('tx.origin'),
                    message: 'Avoid using tx.origin for authorization',
                    suggestion: 'Use msg.sender instead of tx.origin for access control',
                    code: line.trim()
                });
            }

            // Unchecked call
            if (line.match(/\.call\{?.*\}?\(/) && !line.includes('(bool')) {
                issues.push({
                    type: 'UNCHECKED_CALL',
                    severity: 'high',
                    line: lineNum,
                    column: line.indexOf('.call'),
                    message: 'Return value of low-level call not checked',
                    suggestion: 'Check the return value: (bool success, ) = addr.call{...}(...); require(success);',
                    code: line.trim()
                });
            }

            // selfdestruct
            if (line.includes('selfdestruct') && !line.trim().startsWith('//')) {
                issues.push({
                    type: 'SELFDESTRUCT',
                    severity: 'high',
                    line: lineNum,
                    column: line.indexOf('selfdestruct'),
                    message: 'Contract contains selfdestruct - ensure proper access control',
                    suggestion: 'Consider removing selfdestruct or adding strict access control',
                    code: line.trim()
                });
            }

            // delegatecall
            if (line.includes('delegatecall') && !line.trim().startsWith('//')) {
                issues.push({
                    type: 'DELEGATECALL',
                    severity: 'high',
                    line: lineNum,
                    column: line.indexOf('delegatecall'),
                    message: 'delegatecall can be dangerous if target is user-controlled',
                    suggestion: 'Ensure delegatecall target address is trusted and not user-controlled',
                    code: line.trim()
                });
            }

            // block.timestamp in conditions
            if (line.match(/if\s*\(.*block\.timestamp/) || line.match(/require\s*\(.*block\.timestamp/)) {
                issues.push({
                    type: 'TIMESTAMP_DEPENDENCY',
                    severity: 'medium',
                    line: lineNum,
                    column: line.indexOf('block.timestamp'),
                    message: 'block.timestamp can be manipulated by miners',
                    suggestion: 'Avoid using block.timestamp for critical logic. Miners can manipulate it by ~15 seconds.',
                    code: line.trim()
                });
            }
        });

        return issues;
    }
}

// Abstract pattern interface
interface SecurityPattern {
    check(ast: ASTNode, sourceCode: string): SecurityIssue[];
}

// Reentrancy detection
class ReentrancyPattern implements SecurityPattern {
    check(ast: ASTNode, sourceCode: string): SecurityIssue[] {
        const issues: SecurityIssue[] = [];
        const lines = sourceCode.split('\n');

        // Simple heuristic: external call before state change
        this.visitNode(ast, (node) => {
            if (node.type === 'FunctionDefinition') {
                const funcBody = node.body as ASTNode;
                if (!funcBody || !funcBody.statements) return;

                const statements = funcBody.statements as ASTNode[];
                let foundExternalCall = false;
                let externalCallLine = 0;

                for (const stmt of statements) {
                    // Check for external calls
                    if (this.isExternalCall(stmt)) {
                        foundExternalCall = true;
                        externalCallLine = stmt.loc?.start.line || 0;
                    }

                    // Check for state changes after external call
                    if (foundExternalCall && this.isStateChange(stmt)) {
                        issues.push({
                            type: 'REENTRANCY',
                            severity: 'critical',
                            line: externalCallLine,
                            column: 0,
                            message: 'Potential reentrancy vulnerability: state change after external call',
                            suggestion: 'Apply Checks-Effects-Interactions pattern: update state before making external calls',
                            code: lines[externalCallLine - 1]?.trim()
                        });
                    }
                }
            }
        });

        return issues;
    }

    private isExternalCall(node: ASTNode): boolean {
        if (node.type === 'ExpressionStatement') {
            const expr = node.expression as ASTNode;
            if (expr?.type === 'FunctionCall') {
                const callee = expr.expression as ASTNode;
                if (callee?.type === 'MemberAccess') {
                    const member = callee.memberName as string;
                    return ['call', 'send', 'transfer', 'delegatecall'].includes(member);
                }
            }
        }
        return false;
    }

    private isStateChange(node: ASTNode): boolean {
        if (node.type === 'ExpressionStatement') {
            const expr = node.expression as ASTNode;
            if (expr?.type === 'BinaryOperation') {
                const op = expr.operator as string;
                return ['=', '+=', '-=', '*=', '/='].includes(op);
            }
        }
        return false;
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

// Unchecked low-level call
class UncheckedCallPattern implements SecurityPattern {
    check(ast: ASTNode, sourceCode: string): SecurityIssue[] {
        // Implemented in line-based analysis
        return [];
    }
}

// tx.origin authentication
class TxOriginPattern implements SecurityPattern {
    check(ast: ASTNode, sourceCode: string): SecurityIssue[] {
        // Implemented in line-based analysis
        return [];
    }
}

// Integer overflow (pre-0.8.0)
class IntegerOverflowPattern implements SecurityPattern {
    check(ast: ASTNode, sourceCode: string): SecurityIssue[] {
        const issues: SecurityIssue[] = [];

        // Check pragma version
        const pragmaMatch = sourceCode.match(/pragma\s+solidity\s+[\^~]?(\d+)\.(\d+)/);
        if (pragmaMatch) {
            const major = parseInt(pragmaMatch[1]);
            const minor = parseInt(pragmaMatch[2]);

            if (major === 0 && minor < 8) {
                // Check if SafeMath is imported
                if (!sourceCode.includes('SafeMath') && !sourceCode.includes('using SafeMath')) {
                    const lines = sourceCode.split('\n');

                    // Find arithmetic operations
                    lines.forEach((line, index) => {
                        if (line.match(/[\+\-\*](?!=)/) && !line.trim().startsWith('//') && !line.includes('pragma')) {
                            const hasArithmetic = line.match(/\w+\s*[\+\-\*\/]\s*\w+/);
                            if (hasArithmetic) {
                                issues.push({
                                    type: 'INTEGER_OVERFLOW',
                                    severity: 'high',
                                    line: index + 1,
                                    column: 0,
                                    message: 'Potential integer overflow in Solidity < 0.8.0',
                                    suggestion: 'Use SafeMath library or upgrade to Solidity 0.8.0+',
                                    code: line.trim()
                                });
                            }
                        }
                    });
                }
            }
        }

        return issues;
    }
}

// Timestamp dependency
class TimestampDependencyPattern implements SecurityPattern {
    check(ast: ASTNode, sourceCode: string): SecurityIssue[] {
        // Implemented in line-based analysis
        return [];
    }
}

// Dangerous delegatecall
class DelegateCallPattern implements SecurityPattern {
    check(ast: ASTNode, sourceCode: string): SecurityIssue[] {
        // Implemented in line-based analysis
        return [];
    }
}

// Unprotected selfdestruct
class SelfDestructPattern implements SecurityPattern {
    check(ast: ASTNode, sourceCode: string): SecurityIssue[] {
        // Implemented in line-based analysis
        return [];
    }
}

// Arbitrary jump (assembly)
class ArbitraryJumpPattern implements SecurityPattern {
    check(ast: ASTNode, sourceCode: string): SecurityIssue[] {
        const issues: SecurityIssue[] = [];
        const lines = sourceCode.split('\n');

        lines.forEach((line, index) => {
            if (line.includes('assembly') && line.includes('{')) {
                issues.push({
                    type: 'INLINE_ASSEMBLY',
                    severity: 'info',
                    line: index + 1,
                    column: line.indexOf('assembly'),
                    message: 'Inline assembly detected - ensure careful review',
                    suggestion: 'Inline assembly bypasses safety checks. Review carefully for security issues.',
                    code: line.trim()
                });
            }
        });

        return issues;
    }
}

// Unprotected ether withdrawal
class UnprotectedEtherWithdrawalPattern implements SecurityPattern {
    check(ast: ASTNode, sourceCode: string): SecurityIssue[] {
        const issues: SecurityIssue[] = [];
        const lines = sourceCode.split('\n');

        // Look for functions with transfer/send/call that might lack access control
        let inFunction = false;
        let functionLine = 0;
        let hasAccessControl = false;
        let hasEtherTransfer = false;
        let braceCount = 0;

        lines.forEach((line, index) => {
            if (line.match(/function\s+\w+/)) {
                inFunction = true;
                functionLine = index + 1;
                hasAccessControl = false;
                hasEtherTransfer = false;
                braceCount = 0;
            }

            if (inFunction) {
                braceCount += (line.match(/{/g) || []).length;
                braceCount -= (line.match(/}/g) || []).length;

                if (line.includes('onlyOwner') || line.includes('require(msg.sender') || line.includes('require(owner')) {
                    hasAccessControl = true;
                }

                if (line.match(/\.(transfer|send|call\{value)/) && !line.trim().startsWith('//')) {
                    hasEtherTransfer = true;
                }

                if (braceCount === 0 && inFunction) {
                    if (hasEtherTransfer && !hasAccessControl) {
                        issues.push({
                            type: 'UNPROTECTED_WITHDRAWAL',
                            severity: 'critical',
                            line: functionLine,
                            column: 0,
                            message: 'Function with ether transfer may lack access control',
                            suggestion: 'Add access control modifier (e.g., onlyOwner) to functions that transfer ether',
                        });
                    }
                    inFunction = false;
                }
            }
        });

        return issues;
    }
}

// Missing access control
class AccessControlPattern implements SecurityPattern {
    check(ast: ASTNode, sourceCode: string): SecurityIssue[] {
        const issues: SecurityIssue[] = [];
        const lines = sourceCode.split('\n');

        // Check for public/external functions that modify state without access control
        lines.forEach((line, index) => {
            if (line.match(/function\s+\w+.*\b(public|external)\b/) &&
                !line.includes('view') &&
                !line.includes('pure') &&
                !line.includes('onlyOwner') &&
                !line.includes('modifier')) {

                // Check if it's a critical function name
                const criticalPatterns = ['withdraw', 'transfer', 'mint', 'burn', 'pause', 'unpause', 'upgrade', 'set', 'change', 'update'];
                const funcMatch = line.match(/function\s+(\w+)/);

                if (funcMatch) {
                    const funcName = funcMatch[1].toLowerCase();
                    if (criticalPatterns.some(p => funcName.includes(p))) {
                        issues.push({
                            type: 'MISSING_ACCESS_CONTROL',
                            severity: 'medium',
                            line: index + 1,
                            column: 0,
                            message: `Function '${funcMatch[1]}' may need access control`,
                            suggestion: 'Consider adding access control (onlyOwner, role-based) for sensitive functions',
                            code: line.trim()
                        });
                    }
                }
            }
        });

        return issues;
    }
}
