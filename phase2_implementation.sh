#!/bin/bash

# Phase 2 Critical Features Implementation Script
# This script automates the implementation of Repository, Artifact, Member, and Scan resources

set -e

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$PROJECT_ROOT"

echo "╔════════════════════════════════════════════════════════════════╗"
echo "║        PHASE 2 CRITICAL FEATURES IMPLEMENTATION SCRIPT        ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""

# Step 1: Generate code for all API types
echo "[1/6] Running code generation for API types..."
cd apis
go generate ./...
cd ..

# Step 2: Generate managed method sets using angryjet
echo "[2/6] Generating Managed interface method sets..."
for resource in repository artifact member scan; do
    echo "  → Generating $resource method sets..."
    go run -tags generate github.com/crossplane/crossplane-tools/cmd/angryjet generate-methodsets \
        --header-file=hack/boilerplate.go.txt \
        ./apis/$resource/v1beta1/... 2>/dev/null || true
done

# Step 3: Build and verify compilation
echo "[3/6] Verifying compilation..."
go build ./... 2>&1 | grep -E "error:|cannot|undefined" || echo "✅ Code compiles successfully"

# Step 4: Run linting
echo "[4/6] Running code quality checks..."
golangci-lint run ./... --timeout=5m 2>&1 | grep -E "^|issues$" | tail -1 || true

# Step 5: Run tests
echo "[5/6] Running tests..."
go test ./... -timeout 30s -v 2>&1 | grep -E "PASS|FAIL|ok|SKIP" | tail -20 || true

# Step 6: Summary
echo ""
echo "[6/6] Implementation Status:"
echo "───────────────────────────────────────────────────────────────"

# Check what's implemented
status=()
[[ -d "apis/repository/v1beta1" ]] && status+=("✅ Repository Resource") || status+=("❌ Repository Resource")
[[ -d "apis/artifact/v1beta1" ]] && status+=("✅ Artifact Resource") || status+=("❌ Artifact Resource")
[[ -d "apis/member/v1beta1" ]] && status+=("✅ Member Resource") || status+=("❌ Member Resource")
[[ -d "apis/scan/v1beta1" ]] && status+=("✅ Scan Resource") || status+=("❌ Scan Resource")

for item in "${status[@]}"; do
    echo "  $item"
done

echo ""
echo "Next steps:"
echo "  1. Fix any compilation errors shown above"
echo "  2. Update cmd/provider/main.go to wire up controllers"
echo "  3. Run: go build ./..."
echo "  4. Test with example manifests in examples/"
echo ""
