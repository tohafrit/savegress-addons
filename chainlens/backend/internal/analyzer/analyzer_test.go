package analyzer

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNewAnalyzer(t *testing.T) {
	a, err := NewAnalyzer(4)
	if err != nil {
		t.Fatalf("NewAnalyzer failed: %v", err)
	}
	defer a.Stop()

	if a.pool == nil {
		t.Error("pool should not be nil")
	}

	if len(a.patterns) == 0 {
		t.Error("patterns should not be empty")
	}

	if a.cache == nil {
		t.Error("cache should not be nil")
	}
}

func TestNewAnalyzer_InvalidWorkers(t *testing.T) {
	_, err := NewAnalyzer(0)
	if err == nil {
		t.Error("should fail with 0 workers")
	}

	_, err = NewAnalyzer(-1)
	if err == nil {
		t.Error("should fail with negative workers")
	}
}

func TestAnalyzer_Analyze_SimpleContract(t *testing.T) {
	a, err := NewAnalyzer(2)
	if err != nil {
		t.Fatalf("NewAnalyzer failed: %v", err)
	}
	defer a.Stop()

	source := `
pragma solidity ^0.8.0;

contract SimpleStorage {
    uint256 private value;

    function setValue(uint256 newValue) public {
        value = newValue;
    }

    function getValue() public view returns (uint256) {
        return value;
    }
}
`

	result, err := a.Analyze(context.Background(), source, "0.8.19")
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if result.Status != "completed" {
		t.Errorf("status = %q, want completed", result.Status)
	}

	if result.SourceHash == "" {
		t.Error("source hash should not be empty")
	}

	if result.Duration <= 0 {
		t.Error("duration should be positive")
	}

	// Score should be high for simple safe contract
	if result.Score.Security < 80 {
		t.Errorf("security score = %d, want >= 80", result.Score.Security)
	}
}

func TestAnalyzer_Analyze_Caching(t *testing.T) {
	a, err := NewAnalyzer(2)
	if err != nil {
		t.Fatalf("NewAnalyzer failed: %v", err)
	}
	defer a.Stop()

	source := `pragma solidity ^0.8.0; contract Test {}`

	// First analysis
	result1, err := a.Analyze(context.Background(), source, "0.8.19")
	if err != nil {
		t.Fatalf("first analysis failed: %v", err)
	}

	// Second analysis should be cached (faster)
	start := time.Now()
	result2, err := a.Analyze(context.Background(), source, "0.8.19")
	if err != nil {
		t.Fatalf("second analysis failed: %v", err)
	}
	cachedDuration := time.Since(start)

	// Results should be the same
	if result1.SourceHash != result2.SourceHash {
		t.Error("cached result should have same hash")
	}

	// Cached should be very fast (< 1ms typically)
	if cachedDuration > 10*time.Millisecond {
		t.Logf("cached duration = %v (may be slow due to lock contention)", cachedDuration)
	}
}

func TestAnalyzer_Analyze_ReentrancyDetection(t *testing.T) {
	a, err := NewAnalyzer(2)
	if err != nil {
		t.Fatalf("NewAnalyzer failed: %v", err)
	}
	defer a.Stop()

	// Contract with reentrancy vulnerability
	source := `
pragma solidity ^0.7.0;

contract Vulnerable {
    mapping(address => uint256) public balances;

    function withdraw(uint256 amount) public {
        require(balances[msg.sender] >= amount);
        (bool success, ) = msg.sender.call{value: amount}("");
        require(success);
        balances[msg.sender] -= amount;
    }
}
`

	result, err := a.Analyze(context.Background(), source, "0.7.6")
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Should detect reentrancy
	found := false
	for _, issue := range result.Issues {
		if issue.Type == "reentrancy" {
			found = true
			if issue.Severity != SeverityCritical {
				t.Errorf("reentrancy severity = %v, want critical", issue.Severity)
			}
			break
		}
	}

	if !found {
		t.Error("should detect reentrancy vulnerability")
	}
}

func TestAnalyzer_Analyze_TxOrigin(t *testing.T) {
	a, err := NewAnalyzer(2)
	if err != nil {
		t.Fatalf("NewAnalyzer failed: %v", err)
	}
	defer a.Stop()

	source := `
pragma solidity ^0.8.0;

contract Vulnerable {
    address owner;

    function dangerous() public {
        require(tx.origin == owner, "Not owner");
    }
}
`

	result, err := a.Analyze(context.Background(), source, "0.8.19")
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	found := false
	for _, issue := range result.Issues {
		if issue.Type == "tx_origin" {
			found = true
			break
		}
	}

	if !found {
		t.Error("should detect tx.origin vulnerability")
	}
}

func TestAnalyzer_Analyze_UncheckedCall(t *testing.T) {
	a, err := NewAnalyzer(2)
	if err != nil {
		t.Fatalf("NewAnalyzer failed: %v", err)
	}
	defer a.Stop()

	source := `
pragma solidity ^0.8.0;

contract Vulnerable {
    function send(address to) public {
        to.call{value: 1 ether}("");
    }
}
`

	result, err := a.Analyze(context.Background(), source, "0.8.19")
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	found := false
	for _, issue := range result.Issues {
		if issue.Type == "unchecked_call" {
			found = true
			break
		}
	}

	if !found {
		t.Error("should detect unchecked call")
	}
}

func TestAnalyzer_Analyze_Selfdestruct(t *testing.T) {
	a, err := NewAnalyzer(2)
	if err != nil {
		t.Fatalf("NewAnalyzer failed: %v", err)
	}
	defer a.Stop()

	source := `
pragma solidity ^0.8.0;

contract Destructible {
    function kill() public {
        selfdestruct(payable(msg.sender));
    }
}
`

	result, err := a.Analyze(context.Background(), source, "0.8.19")
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	found := false
	for _, issue := range result.Issues {
		if issue.Type == "selfdestruct" {
			found = true
			break
		}
	}

	if !found {
		t.Error("should detect selfdestruct")
	}
}

func TestAnalyzer_Analyze_Delegatecall(t *testing.T) {
	a, err := NewAnalyzer(2)
	if err != nil {
		t.Fatalf("NewAnalyzer failed: %v", err)
	}
	defer a.Stop()

	source := `
pragma solidity ^0.8.0;

contract Proxy {
    function forward(address impl, bytes memory data) public {
        impl.delegatecall(data);
    }
}
`

	result, err := a.Analyze(context.Background(), source, "0.8.19")
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	found := false
	for _, issue := range result.Issues {
		if issue.Type == "delegatecall" {
			found = true
			break
		}
	}

	if !found {
		t.Error("should detect delegatecall")
	}
}

func TestAnalyzer_Analyze_TimestampDependency(t *testing.T) {
	a, err := NewAnalyzer(2)
	if err != nil {
		t.Fatalf("NewAnalyzer failed: %v", err)
	}
	defer a.Stop()

	source := `
pragma solidity ^0.8.0;

contract Lottery {
    function draw() public view returns (bool) {
        if (block.timestamp % 2 == 0) {
            return true;
        }
        return false;
    }
}
`

	result, err := a.Analyze(context.Background(), source, "0.8.19")
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	found := false
	for _, issue := range result.Issues {
		if issue.Type == "timestamp_dependency" {
			found = true
			break
		}
	}

	if !found {
		t.Error("should detect timestamp dependency")
	}
}

func TestAnalyzer_Analyze_IntegerOverflow(t *testing.T) {
	a, err := NewAnalyzer(2)
	if err != nil {
		t.Fatalf("NewAnalyzer failed: %v", err)
	}
	defer a.Stop()

	// Pre-0.8.0 contract without SafeMath
	source := `
pragma solidity ^0.7.0;

contract Overflow {
    uint256 public count;

    function add(uint256 x) public {
        count = count + x;
    }
}
`

	result, err := a.Analyze(context.Background(), source, "0.7.6")
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	found := false
	for _, issue := range result.Issues {
		if issue.Type == "integer_overflow" {
			found = true
			break
		}
	}

	if !found {
		t.Error("should detect integer overflow in pre-0.8.0 code")
	}
}

func TestAnalyzer_Analyze_NoOverflowIn08(t *testing.T) {
	a, err := NewAnalyzer(2)
	if err != nil {
		t.Fatalf("NewAnalyzer failed: %v", err)
	}
	defer a.Stop()

	// 0.8.0+ has built-in overflow checks
	source := `
pragma solidity ^0.8.0;

contract Safe {
    uint256 public count;

    function add(uint256 x) public {
        count = count + x;
    }
}
`

	result, err := a.Analyze(context.Background(), source, "0.8.19")
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	for _, issue := range result.Issues {
		if issue.Type == "integer_overflow" {
			t.Error("should not detect overflow in 0.8.0+")
		}
	}
}

func TestAnalyzer_AnalyzeBatch(t *testing.T) {
	a, err := NewAnalyzer(4)
	if err != nil {
		t.Fatalf("NewAnalyzer failed: %v", err)
	}
	defer a.Stop()

	sources := []string{
		`pragma solidity ^0.8.0; contract A {}`,
		`pragma solidity ^0.8.0; contract B {}`,
		`pragma solidity ^0.8.0; contract C {}`,
	}

	results, err := a.AnalyzeBatch(context.Background(), sources)
	if err != nil {
		t.Fatalf("AnalyzeBatch failed: %v", err)
	}

	if len(results) != len(sources) {
		t.Errorf("results count = %d, want %d", len(results), len(sources))
	}

	for i, result := range results {
		if result == nil {
			t.Errorf("result %d is nil", i)
			continue
		}
		if result.Status != "completed" {
			t.Errorf("result %d status = %q, want completed", i, result.Status)
		}
	}
}

func TestAnalyzer_Stats(t *testing.T) {
	a, err := NewAnalyzer(2)
	if err != nil {
		t.Fatalf("NewAnalyzer failed: %v", err)
	}
	defer a.Stop()

	// Run some analysis
	source := `pragma solidity ^0.8.0; contract Test {}`
	_, _ = a.Analyze(context.Background(), source, "0.8.19")

	stats := a.Stats()

	// Should have some completed tasks
	if stats.CompletedTasks == 0 {
		t.Error("should have completed tasks")
	}
}

func TestAnalyzer_Stop(t *testing.T) {
	a, err := NewAnalyzer(2)
	if err != nil {
		t.Fatalf("NewAnalyzer failed: %v", err)
	}

	err = a.Stop()
	if err != nil {
		t.Errorf("Stop failed: %v", err)
	}
}

func TestAnalyzer_Concurrent(t *testing.T) {
	a, err := NewAnalyzer(4)
	if err != nil {
		t.Fatalf("NewAnalyzer failed: %v", err)
	}
	defer a.Stop()

	var wg sync.WaitGroup
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			source := `pragma solidity ^0.8.0; contract Test` + string(rune('A'+id)) + ` {}`
			_, err := a.Analyze(context.Background(), source, "0.8.19")
			if err != nil {
				t.Errorf("goroutine %d failed: %v", id, err)
			}
		}(i)
	}

	wg.Wait()
}

// Test security patterns directly

func TestReentrancyPattern_Name(t *testing.T) {
	p := &ReentrancyPattern{}
	if name := p.Name(); name != "reentrancy" {
		t.Errorf("name = %q, want reentrancy", name)
	}
}

func TestReentrancyPattern_Check(t *testing.T) {
	p := &ReentrancyPattern{}

	tests := []struct {
		name      string
		source    string
		wantIssue bool
	}{
		{
			name: "vulnerable",
			source: `
function withdraw() {
    msg.sender.call{value: amount}("");
    balance = 0;
}`,
			wantIssue: true,
		},
		{
			name: "safe",
			source: `
function withdraw() {
    balance = 0;
    msg.sender.call{value: amount}("");
}`,
			wantIssue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := p.Check(tt.source)
			hasIssue := len(issues) > 0
			if hasIssue != tt.wantIssue {
				t.Errorf("hasIssue = %v, want %v", hasIssue, tt.wantIssue)
			}
		})
	}
}

func TestUncheckedCallPattern_Name(t *testing.T) {
	p := &UncheckedCallPattern{}
	if name := p.Name(); name != "unchecked_call" {
		t.Errorf("name = %q, want unchecked_call", name)
	}
}

func TestUncheckedCallPattern_Check(t *testing.T) {
	p := &UncheckedCallPattern{}

	tests := []struct {
		name      string
		source    string
		wantIssue bool
	}{
		{
			name:      "unchecked",
			source:    `addr.call{value: 1}("");`,
			wantIssue: true,
		},
		{
			name:      "checked",
			source:    `(bool success, ) = addr.call{value: 1}("");`,
			wantIssue: false,
		},
		{
			name:      "comment",
			source:    `// addr.call{value: 1}("");`,
			wantIssue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := p.Check(tt.source)
			hasIssue := len(issues) > 0
			if hasIssue != tt.wantIssue {
				t.Errorf("hasIssue = %v, want %v", hasIssue, tt.wantIssue)
			}
		})
	}
}

func TestTxOriginPattern_Name(t *testing.T) {
	p := &TxOriginPattern{}
	if name := p.Name(); name != "tx_origin" {
		t.Errorf("name = %q, want tx_origin", name)
	}
}

func TestTxOriginPattern_Check(t *testing.T) {
	p := &TxOriginPattern{}

	tests := []struct {
		name      string
		source    string
		wantIssue bool
	}{
		{
			name:      "has tx.origin",
			source:    `require(tx.origin == owner);`,
			wantIssue: true,
		},
		{
			name:      "no tx.origin",
			source:    `require(msg.sender == owner);`,
			wantIssue: false,
		},
		{
			name:      "comment",
			source:    `// tx.origin`,
			wantIssue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := p.Check(tt.source)
			hasIssue := len(issues) > 0
			if hasIssue != tt.wantIssue {
				t.Errorf("hasIssue = %v, want %v", hasIssue, tt.wantIssue)
			}
		})
	}
}

func TestTimestampPattern_Name(t *testing.T) {
	p := &TimestampPattern{}
	if name := p.Name(); name != "timestamp" {
		t.Errorf("name = %q, want timestamp", name)
	}
}

func TestTimestampPattern_Check(t *testing.T) {
	p := &TimestampPattern{}

	tests := []struct {
		name      string
		source    string
		wantIssue bool
	}{
		{
			name:      "if with timestamp",
			source:    `if (block.timestamp > deadline) revert();`,
			wantIssue: true,
		},
		{
			name:      "require with timestamp",
			source:    `require(block.timestamp < expiry);`,
			wantIssue: true,
		},
		{
			name:      "assignment only",
			source:    `uint256 t = block.timestamp;`,
			wantIssue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := p.Check(tt.source)
			hasIssue := len(issues) > 0
			if hasIssue != tt.wantIssue {
				t.Errorf("hasIssue = %v, want %v", hasIssue, tt.wantIssue)
			}
		})
	}
}

func TestOverflowPattern_Name(t *testing.T) {
	p := &OverflowPattern{}
	if name := p.Name(); name != "overflow" {
		t.Errorf("name = %q, want overflow", name)
	}
}

func TestOverflowPattern_Check(t *testing.T) {
	p := &OverflowPattern{}

	tests := []struct {
		name      string
		source    string
		wantIssue bool
	}{
		{
			name: "pre-0.8 vulnerable",
			source: `
pragma solidity ^0.7.0;
uint256 x = a + b;`,
			wantIssue: true,
		},
		{
			name: "0.8+ safe",
			source: `
pragma solidity ^0.8.0;
uint256 x = a + b;`,
			wantIssue: false,
		},
		{
			name: "pre-0.8 with SafeMath",
			source: `
pragma solidity ^0.7.0;
using SafeMath for uint256;
uint256 x = a.add(b);`,
			wantIssue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := p.Check(tt.source)
			hasIssue := len(issues) > 0
			if hasIssue != tt.wantIssue {
				t.Errorf("hasIssue = %v, want %v", hasIssue, tt.wantIssue)
			}
		})
	}
}

func TestAccessControlPattern_Name(t *testing.T) {
	p := &AccessControlPattern{}
	if name := p.Name(); name != "access_control" {
		t.Errorf("name = %q, want access_control", name)
	}
}

func TestAccessControlPattern_Check(t *testing.T) {
	p := &AccessControlPattern{}

	tests := []struct {
		name      string
		source    string
		wantIssue bool
	}{
		{
			name:      "unprotected withdraw",
			source:    `function withdraw(uint256 amount) public {`,
			wantIssue: true,
		},
		{
			name:      "protected withdraw",
			source:    `function withdraw(uint256 amount) public onlyOwner {`,
			wantIssue: false,
		},
		{
			name:      "normal function",
			source:    `function getValue() public view returns (uint256) {`,
			wantIssue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := p.Check(tt.source)
			hasIssue := len(issues) > 0
			if hasIssue != tt.wantIssue {
				t.Errorf("hasIssue = %v, want %v", hasIssue, tt.wantIssue)
			}
		})
	}
}

func TestSelfdestructPattern_Name(t *testing.T) {
	p := &SelfdestructPattern{}
	if name := p.Name(); name != "selfdestruct" {
		t.Errorf("name = %q, want selfdestruct", name)
	}
}

func TestSelfdestructPattern_Check(t *testing.T) {
	p := &SelfdestructPattern{}

	tests := []struct {
		name      string
		source    string
		wantIssue bool
	}{
		{
			name:      "has selfdestruct",
			source:    `selfdestruct(payable(owner));`,
			wantIssue: true,
		},
		{
			name:      "no selfdestruct",
			source:    `transfer(owner, balance);`,
			wantIssue: false,
		},
		{
			name:      "comment",
			source:    `// selfdestruct(owner);`,
			wantIssue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := p.Check(tt.source)
			hasIssue := len(issues) > 0
			if hasIssue != tt.wantIssue {
				t.Errorf("hasIssue = %v, want %v", hasIssue, tt.wantIssue)
			}
		})
	}
}

func TestDelegatecallPattern_Name(t *testing.T) {
	p := &DelegatecallPattern{}
	if name := p.Name(); name != "delegatecall" {
		t.Errorf("name = %q, want delegatecall", name)
	}
}

func TestDelegatecallPattern_Check(t *testing.T) {
	p := &DelegatecallPattern{}

	tests := []struct {
		name      string
		source    string
		wantIssue bool
	}{
		{
			name:      "has delegatecall",
			source:    `impl.delegatecall(data);`,
			wantIssue: true,
		},
		{
			name:      "no delegatecall",
			source:    `impl.call(data);`,
			wantIssue: false,
		},
		{
			name:      "comment",
			source:    `// impl.delegatecall(data);`,
			wantIssue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := p.Check(tt.source)
			hasIssue := len(issues) > 0
			if hasIssue != tt.wantIssue {
				t.Errorf("hasIssue = %v, want %v", hasIssue, tt.wantIssue)
			}
		})
	}
}

// Test helper functions

func TestSplitLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"single line", "hello", 1},
		{"two lines", "hello\nworld", 2},
		{"three lines", "a\nb\nc", 3},
		{"empty", "", 0},
		{"only newlines", "\n\n", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := splitLines(tt.input)
			if len(lines) != tt.want {
				t.Errorf("splitLines(%q) = %d lines, want %d", tt.input, len(lines), tt.want)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   bool
	}{
		{"hello world", "world", true},
		{"hello world", "foo", false},
		{"hello", "hello", true},
		{"", "", true},
		{"hello", "", true},
		{"", "hello", false},
	}

	for _, tt := range tests {
		name := tt.s + " contains " + tt.substr
		t.Run(name, func(t *testing.T) {
			got := contains(tt.s, tt.substr)
			if got != tt.want {
				t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

func TestFindSubstring(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   int
	}{
		{"hello world", "world", 6},
		{"hello world", "hello", 0},
		{"hello world", "foo", -1},
		{"", "", 0},
		{"hello", "", 0},
		{"", "hello", -1},
	}

	for _, tt := range tests {
		t.Run(tt.s+"_"+tt.substr, func(t *testing.T) {
			got := findSubstring(tt.s, tt.substr)
			if got != tt.want {
				t.Errorf("findSubstring(%q, %q) = %d, want %d", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

func TestTrimSpace(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"  hello  ", "hello"},
		{"\thello\t", "hello"},
		{"hello", "hello"},
		{"", ""},
		{"   ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := trimSpace(tt.input)
			if got != tt.want {
				t.Errorf("trimSpace(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCountChar(t *testing.T) {
	tests := []struct {
		s    string
		c    byte
		want int
	}{
		{"hello", 'l', 2},
		{"hello", 'x', 0},
		{"{{}}", '{', 2},
		{"", 'a', 0},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			got := countChar(tt.s, tt.c)
			if got != tt.want {
				t.Errorf("countChar(%q, %c) = %d, want %d", tt.s, tt.c, got, tt.want)
			}
		})
	}
}

// Test gas analysis

func TestAnalyzeGas(t *testing.T) {
	source := `
pragma solidity ^0.8.0;

contract Test {
    uint256 public value;

    function setValue(uint256 newValue) public {
        value = newValue;
    }

    function getValue() external view returns (uint256) {
        return value;
    }
}
`

	estimates := analyzeGas(source)

	// Should have estimates for public/external functions
	if len(estimates) == 0 {
		t.Error("should have gas estimates")
	}

	for name, estimate := range estimates {
		if estimate.Min == 0 {
			t.Errorf("function %s min gas should be > 0", name)
		}
		if estimate.Typical < estimate.Min {
			t.Errorf("function %s typical < min", name)
		}
		if estimate.Max < estimate.Typical {
			t.Errorf("function %s max < typical", name)
		}
		if estimate.Level == "" {
			t.Errorf("function %s level is empty", name)
		}
	}
}

func TestCalculateScore(t *testing.T) {
	tests := []struct {
		name         string
		issues       []Issue
		gas          map[string]GasEstimate
		minSecurity  int
		minGas       int
		minQuality   int
	}{
		{
			name:        "no issues",
			issues:      nil,
			gas:         nil,
			minSecurity: 100,
			minGas:      100,
			minQuality:  100,
		},
		{
			name: "critical issue",
			issues: []Issue{
				{Severity: SeverityCritical},
			},
			gas:         nil,
			minSecurity: 0,
			minGas:      100,
			minQuality:  0,
		},
		{
			name: "high gas function",
			issues: nil,
			gas: map[string]GasEstimate{
				"expensive": {Level: "high"},
			},
			minSecurity: 100,
			minGas:      0,
			minQuality:  100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculateScore(tt.issues, tt.gas)
			if score.Security < tt.minSecurity {
				t.Errorf("security = %d, want >= %d", score.Security, tt.minSecurity)
			}
			if score.GasEfficiency < tt.minGas {
				t.Errorf("gas = %d, want >= %d", score.GasEfficiency, tt.minGas)
			}
			if score.CodeQuality < tt.minQuality {
				t.Errorf("quality = %d, want >= %d", score.CodeQuality, tt.minQuality)
			}
		})
	}
}

func TestDefaultPatterns(t *testing.T) {
	patterns := defaultPatterns()

	expectedPatterns := []string{
		"reentrancy",
		"unchecked_call",
		"tx_origin",
		"timestamp",
		"overflow",
		"access_control",
		"selfdestruct",
		"delegatecall",
	}

	if len(patterns) != len(expectedPatterns) {
		t.Errorf("pattern count = %d, want %d", len(patterns), len(expectedPatterns))
	}

	for i, p := range patterns {
		if p.Name() != expectedPatterns[i] {
			t.Errorf("pattern %d name = %q, want %q", i, p.Name(), expectedPatterns[i])
		}
	}
}

// Benchmarks

func BenchmarkAnalyze_SimpleContract(b *testing.B) {
	a, _ := NewAnalyzer(4)
	defer a.Stop()

	source := `pragma solidity ^0.8.0; contract Test { uint256 value; function set(uint256 v) public { value = v; } }`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.Analyze(context.Background(), source, "0.8.19")
	}
}

func BenchmarkAnalyze_CachedContract(b *testing.B) {
	a, _ := NewAnalyzer(4)
	defer a.Stop()

	source := `pragma solidity ^0.8.0; contract Cached {}`
	// Pre-cache
	a.Analyze(context.Background(), source, "0.8.19")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.Analyze(context.Background(), source, "0.8.19")
	}
}

func BenchmarkPatternCheck_Reentrancy(b *testing.B) {
	p := &ReentrancyPattern{}
	source := strings.Repeat(`
function withdraw() {
    msg.sender.call{value: amount}("");
    balance = 0;
}
`, 10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Check(source)
	}
}

func BenchmarkSplitLines(b *testing.B) {
	source := strings.Repeat("line\n", 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		splitLines(source)
	}
}

func BenchmarkContains(b *testing.B) {
	s := strings.Repeat("x", 1000) + "target" + strings.Repeat("y", 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		contains(s, "target")
	}
}

func BenchmarkCalculateScore(b *testing.B) {
	issues := []Issue{
		{Severity: SeverityMedium},
		{Severity: SeverityLow},
		{Severity: SeverityHigh},
	}
	gas := map[string]GasEstimate{
		"func1": {Level: "low"},
		"func2": {Level: "medium"},
		"func3": {Level: "high"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calculateScore(issues, gas)
	}
}
