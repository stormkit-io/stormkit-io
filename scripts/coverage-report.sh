#!/bin/bash

# Coverage report script for GitHub Actions
# This script generates a formatted coverage report for PR comments

set -e

COVERAGE_FILE="${1:-coverage.out}"
THRESHOLD="${2:-60.0}"

if [ ! -f "$COVERAGE_FILE" ]; then
    echo "❌ Coverage file not found: $COVERAGE_FILE"
    exit 1
fi

# Calculate total coverage
TOTAL_COVERAGE=$(go tool cover -func="$COVERAGE_FILE" | grep total | awk '{print substr($3, 1, length($3)-1)}')

echo "📊 Coverage Report"
echo "=================="
echo ""
echo "Total Coverage: ${TOTAL_COVERAGE}%"
echo "Threshold: ${THRESHOLD}%"
echo ""

# Check threshold
if (( $(echo "$TOTAL_COVERAGE < $THRESHOLD" | bc -l) )); then
    echo "❌ Coverage ${TOTAL_COVERAGE}% is below threshold ${THRESHOLD}%"
    echo ""
    echo "Packages below threshold:"
    go tool cover -func="$COVERAGE_FILE" | awk -v threshold="$THRESHOLD" '
        $3 != "total:" && substr($3, 1, length($3)-1) < threshold {
            print "  - " $1 ": " $3
        }
    '
    exit 1
else
    echo "✅ Coverage ${TOTAL_COVERAGE}% meets threshold ${THRESHOLD}%"
fi

echo ""
echo "Package Coverage:"
go tool cover -func="$COVERAGE_FILE" | grep -v "total:" | awk '{print "  " $1 ": " $3}'
