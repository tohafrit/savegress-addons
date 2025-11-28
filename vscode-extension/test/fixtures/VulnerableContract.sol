// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

/**
 * @title VulnerableContract
 * @dev This contract contains intentional vulnerabilities for testing ChainLens detection
 */
contract VulnerableContract {
    address public owner;
    mapping(address => uint256) public balances;
    uint256 public totalDeposits;

    event Deposit(address indexed user, uint256 amount);
    event Withdrawal(address indexed user, uint256 amount);

    constructor() {
        owner = msg.sender;
    }

    // VULNERABILITY: Missing access control on sensitive function
    function setOwner(address newOwner) public {
        owner = newOwner;
    }

    function deposit() public payable {
        balances[msg.sender] += msg.value;
        totalDeposits += msg.value;
        emit Deposit(msg.sender, msg.value);
    }

    // VULNERABILITY: Reentrancy - external call before state update
    function withdrawUnsafe(uint256 amount) public {
        require(balances[msg.sender] >= amount, "Insufficient balance");

        // External call BEFORE state update - REENTRANCY RISK!
        (bool success, ) = msg.sender.call{value: amount}("");
        require(success, "Transfer failed");

        // State update AFTER external call
        balances[msg.sender] -= amount;
        totalDeposits -= amount;

        emit Withdrawal(msg.sender, amount);
    }

    // SAFE: Correct implementation following CEI pattern
    function withdrawSafe(uint256 amount) public {
        require(balances[msg.sender] >= amount, "Insufficient balance");

        // State update BEFORE external call
        balances[msg.sender] -= amount;
        totalDeposits -= amount;

        // External call AFTER state update
        (bool success, ) = msg.sender.call{value: amount}("");
        require(success, "Transfer failed");

        emit Withdrawal(msg.sender, amount);
    }

    // VULNERABILITY: Using tx.origin for authorization
    function withdrawToOwner() public {
        require(tx.origin == owner, "Not owner");  // BAD: Use msg.sender instead

        uint256 balance = address(this).balance;
        (bool success, ) = owner.call{value: balance}("");
        require(success, "Transfer failed");
    }

    // VULNERABILITY: Unchecked return value
    function unsafeTransfer(address to, uint256 amount) public {
        require(balances[msg.sender] >= amount, "Insufficient balance");
        balances[msg.sender] -= amount;

        // Return value not checked!
        to.call{value: amount}("");
    }

    // VULNERABILITY: Block timestamp dependency
    function timeLock() public view returns (bool) {
        // BAD: Miners can manipulate block.timestamp
        if (block.timestamp > 1700000000) {
            return true;
        }
        return false;
    }

    // HIGH GAS: Multiple storage writes
    function batchUpdate(address[] memory users, uint256[] memory amounts) public {
        require(users.length == amounts.length, "Length mismatch");

        for (uint256 i = 0; i < users.length; i++) {
            balances[users[i]] = amounts[i];  // Storage write in loop!
            totalDeposits += amounts[i];       // Another storage write!
        }
    }

    // VULNERABILITY: selfdestruct without proper access control
    function destroy() public {
        selfdestruct(payable(owner));
    }

    // Using inline assembly
    function getBalance() public view returns (uint256 result) {
        assembly {
            result := selfbalance()
        }
    }

    receive() external payable {
        balances[msg.sender] += msg.value;
    }
}

// Interface example
interface IVault {
    function deposit() external payable;
    function withdraw(uint256 amount) external;
    function balanceOf(address account) external view returns (uint256);
}

// Library example
library SafeMath {
    function add(uint256 a, uint256 b) internal pure returns (uint256) {
        uint256 c = a + b;
        require(c >= a, "SafeMath: addition overflow");
        return c;
    }

    function sub(uint256 a, uint256 b) internal pure returns (uint256) {
        require(b <= a, "SafeMath: subtraction overflow");
        return a - b;
    }
}
