import * as assert from 'assert';
import { GasAnalyzer, GasEstimate, GasReport } from '../analyzers/gas';

suite('GasAnalyzer Test Suite', () => {
    let analyzer: GasAnalyzer;

    setup(() => {
        analyzer = new GasAnalyzer();
    });

    suite('Function Gas Estimation', () => {
        test('should estimate gas for simple function', () => {
            const code = `
                contract Simple {
                    function simple() public pure returns (uint256) {
                        return 42;
                    }
                }
            `;

            const estimates = analyzer.analyze(code);

            assert.ok(estimates.length > 0, 'Should return estimates');
            const simple = estimates.find(e => e.name === 'simple');
            assert.ok(simple, 'Should find simple function');
            assert.strictEqual(simple?.level, 'low', 'Simple function should have low gas');
        });

        test('should estimate higher gas for storage operations', () => {
            const code = `
                contract Storage {
                    uint256 public value;

                    function setValue(uint256 newValue) public {
                        value = newValue;
                    }
                }
            `;

            const estimates = analyzer.analyze(code);
            const setValue = estimates.find(e => e.name === 'setValue');

            assert.ok(setValue, 'Should find setValue function');
            assert.ok(setValue!.estimatedGas > 21000, 'Storage write should increase gas');
        });

        test('should estimate gas for function with external call', () => {
            const code = `
                contract WithCall {
                    function sendEther(address to) public payable {
                        (bool success, ) = to.call{value: msg.value}("");
                        require(success);
                    }
                }
            `;

            const estimates = analyzer.analyze(code);
            const sendEther = estimates.find(e => e.name === 'sendEther');

            assert.ok(sendEther, 'Should find sendEther function');
            assert.ok(sendEther!.estimatedGas >= 23600, 'External call should add ~2600 gas');
        });

        test('should flag high gas for loops', () => {
            const code = `
                contract WithLoop {
                    uint256[] public values;

                    function batchUpdate(uint256[] memory newValues) public {
                        for (uint256 i = 0; i < newValues.length; i++) {
                            values.push(newValues[i]);
                        }
                    }
                }
            `;

            const estimates = analyzer.analyze(code);
            const batchUpdate = estimates.find(e => e.name === 'batchUpdate');

            assert.ok(batchUpdate, 'Should find batchUpdate function');
            assert.ok(
                batchUpdate!.suggestions.some(s => s.toLowerCase().includes('loop')),
                'Should warn about loops'
            );
        });

        test('should suggest external over public', () => {
            const code = `
                contract Public {
                    function doSomething() public pure returns (uint256) {
                        return 1;
                    }
                }
            `;

            const estimates = analyzer.analyze(code);
            const doSomething = estimates.find(e => e.name === 'doSomething');

            assert.ok(doSomething, 'Should find doSomething function');
            assert.ok(
                doSomething!.suggestions.some(s => s.includes('external')),
                'Should suggest using external instead of public'
            );
        });
    });

    suite('Gas Report', () => {
        test('should generate complete gas report', () => {
            const code = `
                pragma solidity ^0.8.19;

                contract Example {
                    uint256 public counter;

                    constructor() {
                        counter = 0;
                    }

                    function increment() public {
                        counter += 1;
                    }

                    function decrement() public {
                        counter -= 1;
                    }
                }
            `;

            const report = analyzer.generateReport(code);

            assert.ok(report.functions.length >= 2, 'Should analyze multiple functions');
            assert.ok(report.totalDeploymentGas > 0, 'Should estimate deployment gas');
            assert.ok(Array.isArray(report.suggestions), 'Should include suggestions');
        });

        test('should estimate deployment gas based on code size', () => {
            const smallCode = `
                contract Small {
                    function foo() public pure returns (uint256) { return 1; }
                }
            `;

            const largeCode = `
                contract Large {
                    uint256 public a;
                    uint256 public b;
                    uint256 public c;
                    mapping(address => uint256) public balances;

                    function foo() public { a = 1; }
                    function bar() public { b = 2; }
                    function baz() public { c = 3; }
                    function deposit() public payable { balances[msg.sender] += msg.value; }
                    function withdraw() public { payable(msg.sender).transfer(balances[msg.sender]); }
                }
            `;

            const smallReport = analyzer.generateReport(smallCode);
            const largeReport = analyzer.generateReport(largeCode);

            assert.ok(
                largeReport.totalDeploymentGas > smallReport.totalDeploymentGas,
                'Larger contract should have higher deployment gas'
            );
        });
    });

    suite('Optimization Suggestions', () => {
        test('should suggest bytes32 over string', () => {
            const code = `
                contract WithString {
                    string public name;

                    function setName(string memory _name) public {
                        name = _name;
                    }
                }
            `;

            const report = analyzer.generateReport(code);

            assert.ok(
                report.suggestions.some(s => s.includes('bytes32') && s.includes('string')),
                'Should suggest bytes32 over string'
            );
        });

        test('should suggest uint256 over smaller uint types', () => {
            const code = `
                contract SmallInts {
                    uint8 public small;
                    uint16 public medium;

                    function set(uint8 _small, uint16 _medium) public {
                        small = _small;
                        medium = _medium;
                    }
                }
            `;

            const report = analyzer.generateReport(code);

            assert.ok(
                report.suggestions.some(s => s.includes('uint8') || s.includes('uint16')),
                'Should warn about small uint types'
            );
        });

        test('should suggest ++i over i++', () => {
            const code = `
                contract Loop {
                    function count() public pure returns (uint256) {
                        uint256 sum = 0;
                        for (uint256 i = 0; i < 10; i++) {
                            sum += i;
                        }
                        return sum;
                    }
                }
            `;

            const report = analyzer.generateReport(code);

            assert.ok(
                report.suggestions.some(s => s.includes('++i')),
                'Should suggest ++i instead of i++'
            );
        });

        test('should suggest custom errors', () => {
            const code = `
                pragma solidity ^0.8.19;

                contract MultiRequire {
                    function check(uint256 a, uint256 b, uint256 c, uint256 d, uint256 e, uint256 f) public pure {
                        require(a > 0, "a");
                        require(b > 0, "b");
                        require(c > 0, "c");
                        require(d > 0, "d");
                        require(e > 0, "e");
                        require(f > 0, "f");
                    }
                }
            `;

            const report = analyzer.generateReport(code);

            assert.ok(
                report.suggestions.some(s => s.includes('custom errors')),
                'Should suggest custom errors for multiple requires'
            );
        });

        test('should suggest unchecked blocks for 0.8+', () => {
            const code = `
                pragma solidity ^0.8.19;

                contract NoUnchecked {
                    function add(uint256 a, uint256 b) public pure returns (uint256) {
                        return a + b;
                    }
                }
            `;

            const report = analyzer.generateReport(code);

            assert.ok(
                report.suggestions.some(s => s.includes('unchecked')),
                'Should suggest unchecked blocks for Solidity 0.8+'
            );
        });
    });

    suite('Gas Level Classification', () => {
        test('should classify low gas functions correctly', () => {
            const code = `
                contract View {
                    uint256 public value = 42;

                    function getValue() public view returns (uint256) {
                        return value;
                    }
                }
            `;

            const estimates = analyzer.analyze(code);
            const getValue = estimates.find(e => e.name === 'getValue');

            assert.ok(getValue, 'Should find getValue function');
            assert.strictEqual(getValue?.level, 'low', 'View function should be low gas');
        });

        test('should classify high gas functions correctly', () => {
            const code = `
                contract Heavy {
                    mapping(address => uint256) balances;
                    event Transfer(address indexed from, address indexed to, uint256 amount);

                    function heavyOperation(address[] memory addresses, uint256[] memory amounts) public {
                        for (uint256 i = 0; i < addresses.length; i++) {
                            balances[addresses[i]] = amounts[i];
                            emit Transfer(msg.sender, addresses[i], amounts[i]);
                        }
                    }
                }
            `;

            const estimates = analyzer.analyze(code);
            const heavy = estimates.find(e => e.name === 'heavyOperation');

            assert.ok(heavy, 'Should find heavyOperation function');
            // With loops and multiple storage writes, should be medium or high
            assert.ok(
                heavy?.level === 'medium' || heavy?.level === 'high',
                `Heavy function should be medium or high gas, got ${heavy?.level}`
            );
        });
    });

    suite('Edge Cases', () => {
        test('should handle interface functions', () => {
            const code = `
                interface IToken {
                    function transfer(address to, uint256 amount) external returns (bool);
                    function balanceOf(address account) external view returns (uint256);
                }
            `;

            const estimates = analyzer.analyze(code);
            // Interface functions have no body, should not crash
            assert.ok(Array.isArray(estimates), 'Should return array for interface');
        });

        test('should handle library functions', () => {
            const code = `
                library SafeMath {
                    function add(uint256 a, uint256 b) internal pure returns (uint256) {
                        uint256 c = a + b;
                        require(c >= a, "overflow");
                        return c;
                    }
                }
            `;

            const estimates = analyzer.analyze(code);
            const add = estimates.find(e => e.name === 'add');

            assert.ok(add, 'Should analyze library function');
        });

        test('should handle constructor', () => {
            const code = `
                contract WithConstructor {
                    address public owner;

                    constructor() {
                        owner = msg.sender;
                    }
                }
            `;

            // Constructor is not a named function, analyzer may or may not include it
            const estimates = analyzer.analyze(code);
            assert.ok(Array.isArray(estimates), 'Should handle constructor without crashing');
        });

        test('should handle empty contract', () => {
            const code = `
                contract Empty {
                }
            `;

            const estimates = analyzer.analyze(code);
            assert.ok(Array.isArray(estimates), 'Should handle empty contract');
            assert.strictEqual(estimates.length, 0, 'Empty contract should have no function estimates');
        });

        test('should handle malformed code gracefully', () => {
            const code = `
                contract Broken {
                    function broken( {
                        this is not valid solidity
                    }
                }
            `;

            // Should not throw, should use fallback analysis
            const estimates = analyzer.analyze(code);
            assert.ok(Array.isArray(estimates), 'Should return array even for malformed code');
        });
    });
});
