#!/bin/bash

# ChainLens Benchmark Script
# Usage: ./scripts/benchmark.sh [options]
#
# Options:
#   -o, --output FILE    Save results to file (default: stdout)
#   -c, --compare FILE   Compare with previous results
#   -p, --package PKG    Run benchmarks only for specific package
#   -t, --time SECONDS   Benchmark time per test (default: 3s)
#   -m, --memory         Include memory profiling
#   -h, --help           Show this help

set -e

OUTPUT=""
COMPARE=""
PACKAGE="./..."
BENCHTIME="3s"
MEMPROFILE=""

while [[ $# -gt 0 ]]; do
    case $1 in
        -o|--output)
            OUTPUT="$2"
            shift 2
            ;;
        -c|--compare)
            COMPARE="$2"
            shift 2
            ;;
        -p|--package)
            PACKAGE="$2"
            shift 2
            ;;
        -t|--time)
            BENCHTIME="$2"
            shift 2
            ;;
        -m|--memory)
            MEMPROFILE="1"
            shift
            ;;
        -h|--help)
            head -15 "$0" | tail -13
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

echo "============================================"
echo "ChainLens Benchmark Suite"
echo "============================================"
echo "Package: $PACKAGE"
echo "Benchmark time: $BENCHTIME"
echo "Date: $(date)"
echo "Go version: $(go version)"
echo "============================================"
echo ""

# Build flags for benchmarks
FLAGS="-bench=. -benchtime=$BENCHTIME"

if [ -n "$MEMPROFILE" ]; then
    FLAGS="$FLAGS -benchmem"
fi

# Run benchmarks
if [ -n "$OUTPUT" ]; then
    echo "Running benchmarks and saving to $OUTPUT..."
    go test $PACKAGE $FLAGS -count=1 2>&1 | tee "$OUTPUT"
else
    echo "Running benchmarks..."
    go test $PACKAGE $FLAGS -count=1
fi

# Compare with previous results if requested
if [ -n "$COMPARE" ] && [ -n "$OUTPUT" ]; then
    if command -v benchstat &> /dev/null; then
        echo ""
        echo "============================================"
        echo "Comparison with previous results:"
        echo "============================================"
        benchstat "$COMPARE" "$OUTPUT"
    else
        echo ""
        echo "Install benchstat for comparison: go install golang.org/x/perf/cmd/benchstat@latest"
    fi
fi

echo ""
echo "Benchmark complete!"
