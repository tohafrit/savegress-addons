import * as assert from 'assert';
import { SecurityAnalyzer } from '../analyzers/security';

suite('SecurityAnalyzer Test Suite', () => {
    let analyzer: SecurityAnalyzer;

    setup(() => {
        analyzer = new SecurityAnalyzer();
    });

    suite('Reentrancy Detection', () => {
        test('should detect reentrancy when external call before state update', () => {
            const code = `
                contract Vulnerable {
                    mapping(address => uint256) balances;

                    function withdrawUnsafe(uint256 amount) public {
                        require(balances[msg.sender] >= amount);
                        (bool success, ) = msg.sender.call{value: amount}("");
                        require(success);
                        balances[msg.sender] -= amount;
                    }
                }
            `;

            const issues = analyzer.analyze(code);
            const reentrancy = issues.find(i => i.type === 'REENTRANCY');

            assert.ok(reentrancy, 'Should detect reentrancy vulnerability');
            assert.strictEqual(reentrancy?.severity, 'critical');
        });

        test('should not flag safe CEI pattern', () => {
            const code = `
                contract Safe {
                    mapping(address => uint256) balances;

                    function withdrawSafe(uint256 amount) public {
                        require(balances[msg.sender] >= amount);
                        balances[msg.sender] -= amount;
                        (bool success, ) = msg.sender.call{value: amount}("");
                        require(success);
                    }
                }
            `;

            const issues = analyzer.analyze(code);
            const reentrancy = issues.find(i => i.type === 'REENTRANCY');

            assert.ok(!reentrancy, 'Should not flag safe CEI pattern as reentrancy');
        });
    });

    suite('tx.origin Detection', () => {
        test('should detect tx.origin usage', () => {
            const code = `
                contract Vulnerable {
                    address owner;

                    function withdraw() public {
                        require(tx.origin == owner);
                        payable(owner).transfer(address(this).balance);
                    }
                }
            `;

            const issues = analyzer.analyze(code);
            const txOrigin = issues.find(i => i.type === 'TX_ORIGIN_AUTH');

            assert.ok(txOrigin, 'Should detect tx.origin usage');
            assert.strictEqual(txOrigin?.severity, 'high');
        });

        test('should not flag tx.origin in comments', () => {
            const code = `
                contract Safe {
                    // Don't use tx.origin for auth
                    function safe() public view returns (address) {
                        return msg.sender;
                    }
                }
            `;

            const issues = analyzer.analyze(code);
            const txOrigin = issues.find(i => i.type === 'TX_ORIGIN_AUTH');

            assert.ok(!txOrigin, 'Should not flag tx.origin in comments');
        });
    });

    suite('Unchecked Call Detection', () => {
        test('should detect unchecked low-level call', () => {
            const code = `
                contract Vulnerable {
                    function unsafeTransfer(address to) public {
                        to.call{value: 1 ether}("");
                    }
                }
            `;

            const issues = analyzer.analyze(code);
            const unchecked = issues.find(i => i.type === 'UNCHECKED_CALL');

            assert.ok(unchecked, 'Should detect unchecked call');
            assert.strictEqual(unchecked?.severity, 'high');
        });

        test('should not flag checked call', () => {
            const code = `
                contract Safe {
                    function safeTransfer(address to) public {
                        (bool success, ) = to.call{value: 1 ether}("");
                        require(success);
                    }
                }
            `;

            const issues = analyzer.analyze(code);
            const unchecked = issues.find(i => i.type === 'UNCHECKED_CALL');

            assert.ok(!unchecked, 'Should not flag checked call');
        });
    });

    suite('selfdestruct Detection', () => {
        test('should detect selfdestruct', () => {
            const code = `
                contract Vulnerable {
                    function destroy() public {
                        selfdestruct(payable(msg.sender));
                    }
                }
            `;

            const issues = analyzer.analyze(code);
            const selfDestruct = issues.find(i => i.type === 'SELFDESTRUCT');

            assert.ok(selfDestruct, 'Should detect selfdestruct');
            assert.strictEqual(selfDestruct?.severity, 'high');
        });
    });

    suite('delegatecall Detection', () => {
        test('should detect delegatecall', () => {
            const code = `
                contract Vulnerable {
                    function execute(address target, bytes memory data) public {
                        target.delegatecall(data);
                    }
                }
            `;

            const issues = analyzer.analyze(code);
            const delegatecall = issues.find(i => i.type === 'DELEGATECALL');

            assert.ok(delegatecall, 'Should detect delegatecall');
            assert.strictEqual(delegatecall?.severity, 'high');
        });
    });

    suite('Timestamp Dependency Detection', () => {
        test('should detect block.timestamp in condition', () => {
            const code = `
                contract Vulnerable {
                    function timeLock() public view returns (bool) {
                        if (block.timestamp > 1700000000) {
                            return true;
                        }
                        return false;
                    }
                }
            `;

            const issues = analyzer.analyze(code);
            const timestamp = issues.find(i => i.type === 'TIMESTAMP_DEPENDENCY');

            assert.ok(timestamp, 'Should detect timestamp dependency');
            assert.strictEqual(timestamp?.severity, 'medium');
        });
    });

    suite('Inline Assembly Detection', () => {
        test('should detect inline assembly', () => {
            const code = `
                contract WithAssembly {
                    function getBalance() public view returns (uint256 result) {
                        assembly {
                            result := selfbalance()
                        }
                    }
                }
            `;

            const issues = analyzer.analyze(code);
            const assembly = issues.find(i => i.type === 'INLINE_ASSEMBLY');

            assert.ok(assembly, 'Should detect inline assembly');
            assert.strictEqual(assembly?.severity, 'info');
        });
    });

    suite('Access Control Detection', () => {
        test('should detect missing access control on sensitive function', () => {
            const code = `
                contract Vulnerable {
                    address public owner;

                    function setOwner(address newOwner) public {
                        owner = newOwner;
                    }
                }
            `;

            const issues = analyzer.analyze(code);
            const accessControl = issues.find(i => i.type === 'MISSING_ACCESS_CONTROL');

            assert.ok(accessControl, 'Should detect missing access control');
            assert.strictEqual(accessControl?.severity, 'medium');
        });

        test('should not flag function with onlyOwner modifier', () => {
            const code = `
                contract Safe {
                    address public owner;

                    modifier onlyOwner() {
                        require(msg.sender == owner);
                        _;
                    }

                    function setOwner(address newOwner) public onlyOwner {
                        owner = newOwner;
                    }
                }
            `;

            const issues = analyzer.analyze(code);
            const accessControl = issues.find(i =>
                i.type === 'MISSING_ACCESS_CONTROL' &&
                i.message.includes('setOwner')
            );

            assert.ok(!accessControl, 'Should not flag function with onlyOwner');
        });
    });

    suite('Unprotected Withdrawal Detection', () => {
        test('should detect unprotected ether withdrawal', () => {
            const code = `
                contract Vulnerable {
                    function withdraw() public {
                        payable(msg.sender).transfer(address(this).balance);
                    }
                }
            `;

            const issues = analyzer.analyze(code);
            const unprotected = issues.find(i => i.type === 'UNPROTECTED_WITHDRAWAL');

            assert.ok(unprotected, 'Should detect unprotected withdrawal');
            assert.strictEqual(unprotected?.severity, 'critical');
        });

        test('should not flag protected withdrawal', () => {
            const code = `
                contract Safe {
                    address owner;

                    function withdraw() public {
                        require(msg.sender == owner);
                        payable(owner).transfer(address(this).balance);
                    }
                }
            `;

            const issues = analyzer.analyze(code);
            const unprotected = issues.find(i => i.type === 'UNPROTECTED_WITHDRAWAL');

            assert.ok(!unprotected, 'Should not flag protected withdrawal');
        });
    });

    suite('Issue Sorting', () => {
        test('should sort issues by severity', () => {
            const code = `
                contract Mixed {
                    function bad() public {
                        selfdestruct(payable(msg.sender));
                        if (block.timestamp > 0) {}
                    }

                    function alsobad() public {
                        payable(msg.sender).transfer(1 ether);
                    }
                }
            `;

            const issues = analyzer.analyze(code);

            // Verify issues are sorted by severity
            const severityOrder = { critical: 0, high: 1, medium: 2, low: 3, info: 4 };
            for (let i = 1; i < issues.length; i++) {
                const prevOrder = severityOrder[issues[i - 1].severity];
                const currOrder = severityOrder[issues[i].severity];
                assert.ok(prevOrder <= currOrder, 'Issues should be sorted by severity');
            }
        });
    });

    suite('Real-world Vulnerable Contract', () => {
        test('should detect multiple vulnerabilities in test fixture', () => {
            // This is the VulnerableContract.sol from fixtures
            const code = `
                pragma solidity ^0.8.19;

                contract VulnerableContract {
                    address public owner;
                    mapping(address => uint256) public balances;

                    function setOwner(address newOwner) public {
                        owner = newOwner;
                    }

                    function withdrawUnsafe(uint256 amount) public {
                        require(balances[msg.sender] >= amount);
                        (bool success, ) = msg.sender.call{value: amount}("");
                        require(success);
                        balances[msg.sender] -= amount;
                    }

                    function withdrawToOwner() public {
                        require(tx.origin == owner);
                        (bool success, ) = owner.call{value: address(this).balance}("");
                        require(success);
                    }

                    function unsafeTransfer(address to, uint256 amount) public {
                        to.call{value: amount}("");
                    }

                    function timeLock() public view returns (bool) {
                        if (block.timestamp > 1700000000) {
                            return true;
                        }
                        return false;
                    }

                    function destroy() public {
                        selfdestruct(payable(owner));
                    }

                    function getBalance() public view returns (uint256 result) {
                        assembly {
                            result := selfbalance()
                        }
                    }
                }
            `;

            const issues = analyzer.analyze(code);

            // Should detect multiple issue types
            const issueTypes = new Set(issues.map(i => i.type));

            assert.ok(issueTypes.has('REENTRANCY'), 'Should detect reentrancy');
            assert.ok(issueTypes.has('TX_ORIGIN_AUTH'), 'Should detect tx.origin');
            assert.ok(issueTypes.has('UNCHECKED_CALL'), 'Should detect unchecked call');
            assert.ok(issueTypes.has('TIMESTAMP_DEPENDENCY'), 'Should detect timestamp dependency');
            assert.ok(issueTypes.has('SELFDESTRUCT'), 'Should detect selfdestruct');
            assert.ok(issueTypes.has('INLINE_ASSEMBLY'), 'Should detect inline assembly');
            assert.ok(issueTypes.has('MISSING_ACCESS_CONTROL'), 'Should detect missing access control');

            // Should find at least 7 issues
            assert.ok(issues.length >= 7, `Should find at least 7 issues, found ${issues.length}`);
        });
    });
});
