#!/bin/bash

# Mercury Relay Test Runner
# This script runs all tests for the Mercury Relay project

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
TEST_TIMEOUT="10m"
COVERAGE_THRESHOLD=80
VERBOSE=false
INTEGRATION_TESTS=false
STRESS_TESTS=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -i|--integration)
            INTEGRATION_TESTS=true
            shift
            ;;
        -s|--stress)
            STRESS_TESTS=true
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo "Options:"
            echo "  -v, --verbose      Run tests in verbose mode"
            echo "  -i, --integration  Run integration tests (requires Docker)"
            echo "  -s, --stress       Run stress tests"
            echo "  -h, --help         Show this help message"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to check prerequisites
check_prerequisites() {
    print_status "Checking prerequisites..."
    
    if ! command_exists go; then
        print_error "Go is not installed. Please install Go 1.21 or later."
        exit 1
    fi
    
    if ! command_exists docker; then
        print_warning "Docker is not installed. Integration tests will be skipped."
        INTEGRATION_TESTS=false
    fi
    
    if ! command_exists docker-compose; then
        print_warning "Docker Compose is not installed. Integration tests will be skipped."
        INTEGRATION_TESTS=false
    fi
    
    print_success "Prerequisites check completed"
}

# Function to run unit tests
run_unit_tests() {
    print_status "Running unit tests..."
    
    local test_args="-timeout=$TEST_TIMEOUT"
    
    if [ "$VERBOSE" = true ]; then
        test_args="$test_args -v"
    fi
    
    # Run tests with coverage
    go test $test_args -coverprofile=coverage.out -covermode=atomic ./...
    
    if [ $? -eq 0 ]; then
        print_success "Unit tests passed"
    else
        print_error "Unit tests failed"
        exit 1
    fi
}

# Function to run integration tests
run_integration_tests() {
    if [ "$INTEGRATION_TESTS" = false ]; then
        print_status "Skipping integration tests (use -i to enable)"
        return
    fi
    
    print_status "Running integration tests..."
    
    # Start test services
    print_status "Starting test services..."
    docker-compose -f docker-compose.test.yml up -d
    
    # Wait for services to be ready
    print_status "Waiting for services to be ready..."
    sleep 30
    
    # Run integration tests
    local test_args="-timeout=$TEST_TIMEOUT -tags=integration"
    
    if [ "$VERBOSE" = true ]; then
        test_args="$test_args -v"
    fi
    
    go test $test_args ./test/integration/...
    
    if [ $? -eq 0 ]; then
        print_success "Integration tests passed"
    else
        print_error "Integration tests failed"
        docker-compose -f docker-compose.test.yml down
        exit 1
    fi
    
    # Stop test services
    print_status "Stopping test services..."
    docker-compose -f docker-compose.test.yml down
}

# Function to run stress tests
run_stress_tests() {
    if [ "$STRESS_TESTS" = false ]; then
        print_status "Skipping stress tests (use -s to enable)"
        return
    fi
    
    print_status "Running stress tests..."
    
    local test_args="-timeout=$TEST_TIMEOUT -tags=stress"
    
    if [ "$VERBOSE" = true ]; then
        test_args="$test_args -v"
    fi
    
    go test $test_args ./test/integration/ -run=TestStressTest
    
    if [ $? -eq 0 ]; then
        print_success "Stress tests passed"
    else
        print_error "Stress tests failed"
        exit 1
    fi
}

# Function to generate coverage report
generate_coverage_report() {
    print_status "Generating coverage report..."
    
    if [ ! -f coverage.out ]; then
        print_error "Coverage file not found. Run unit tests first."
        return
    fi
    
    # Generate HTML coverage report
    go tool cover -html=coverage.out -o coverage.html
    
    # Generate coverage summary
    coverage_percent=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
    
    print_status "Coverage: ${coverage_percent}%"
    
    if (( $(echo "$coverage_percent < $COVERAGE_THRESHOLD" | bc -l) )); then
        print_warning "Coverage is below threshold of ${COVERAGE_THRESHOLD}%"
    else
        print_success "Coverage meets threshold of ${COVERAGE_THRESHOLD}%"
    fi
    
    print_status "Coverage report generated: coverage.html"
}

# Function to run benchmarks
run_benchmarks() {
    print_status "Running benchmarks..."
    
    local bench_args="-bench=."
    
    if [ "$VERBOSE" = true ]; then
        bench_args="$bench_args -v"
    fi
    
    go test $bench_args -benchmem ./...
    
    if [ $? -eq 0 ]; then
        print_success "Benchmarks completed"
    else
        print_error "Benchmarks failed"
        exit 1
    fi
}

# Function to run linting
run_linting() {
    print_status "Running linting..."
    
    if command_exists golangci-lint; then
        golangci-lint run
        if [ $? -eq 0 ]; then
            print_success "Linting passed"
        else
            print_error "Linting failed"
            exit 1
        fi
    else
        print_warning "golangci-lint not found. Skipping linting."
        print_status "Install golangci-lint for better code quality checks:"
        print_status "  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b \$(go env GOPATH)/bin v1.54.2"
    fi
}

# Function to run security checks
run_security_checks() {
    print_status "Running security checks..."
    
    if command_exists gosec; then
        gosec ./...
        if [ $? -eq 0 ]; then
            print_success "Security checks passed"
        else
            print_error "Security checks failed"
            exit 1
        fi
    else
        print_warning "gosec not found. Skipping security checks."
        print_status "Install gosec for security analysis:"
        print_status "  go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"
    fi
}

# Function to clean up test artifacts
cleanup() {
    print_status "Cleaning up test artifacts..."
    
    # Remove coverage files
    rm -f coverage.out coverage.html
    
    # Stop any running test services
    docker-compose -f docker-compose.test.yml down >/dev/null 2>&1 || true
    
    print_success "Cleanup completed"
}

# Main execution
main() {
    print_status "Starting Mercury Relay test suite..."
    
    # Check prerequisites
    check_prerequisites
    
    # Run linting first
    run_linting
    
    # Run security checks
    run_security_checks
    
    # Run unit tests
    run_unit_tests
    
    # Run integration tests
    run_integration_tests
    
    # Run stress tests
    run_stress_tests
    
    # Run benchmarks
    run_benchmarks
    
    # Generate coverage report
    generate_coverage_report
    
    print_success "All tests completed successfully!"
    
    # Cleanup
    cleanup
}

# Trap to ensure cleanup on exit
trap cleanup EXIT

# Run main function
main
