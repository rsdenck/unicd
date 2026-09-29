#!/usr/bin/env bash
#
# Release script for unicd CLI
#
# Usage:
#   ./scripts/release.sh [version]
#
# Examples:
#   ./scripts/release.sh v1.5.0
#   ./scripts/release.sh v1.5.0-beta.1
#

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Functions
log_info() {
    echo -e "${BLUE}==>${NC} $*"
}

log_success() {
    echo -e "${GREEN}✓${NC} $*"
}

log_error() {
    echo -e "${RED}✗${NC} $*" >&2
}

log_warn() {
    echo -e "${YELLOW}!${NC} $*"
}

# Get version from argument or git
get_version() {
    if [ -n "${1:-}" ]; then
        echo "$1"
    else
        # Try to get the latest tag and increment patch
        latest_tag=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
        version="${latest_tag#v}"
        IFS='.' read -r major minor patch <<< "$version"
        new_patch=$((patch + 1))
        echo "v${major}.${minor}.${new_patch}"
    fi
}

# Validate version format
validate_version() {
    version="$1"
    if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9]+)?$ ]]; then
        log_error "Invalid version format: $version"
        log_error "Expected format: v1.2.3 or v1.2.3-beta.1"
        exit 1
    fi
}

# Check if we're on the main branch
check_branch() {
    branch=$(git rev-parse --abbrev-ref HEAD)
    if [ "$branch" != "main" ] && [ "$branch" != "master" ]; then
        log_warn "Not on main/master branch (current: $branch)"
        read -p "Continue anyway? [y/N] " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    fi
}

# Check for uncommitted changes
check_clean() {
    if [ -n "$(git status --porcelain)" ]; then
        log_error "Working directory is not clean"
        git status --short
        exit 1
    fi
}

# Run tests
run_tests() {
    log_info "Running tests..."
    
    log_info "Checking formatting..."
    unformatted="$(gofmt -l .)"
    if [ -n "$unformatted" ]; then
        log_error "Files not formatted:"
        echo "$unformatted"
        exit 1
    fi
    
    log_info "Running go vet..."
    go vet ./...
    
    log_info "Running tests..."
    go test ./...
    
    log_info "Building..."
    go build -trimpath -o unicd .
    
    log_info "Smoke test..."
    ./unicd --no-banner version
    
    log_success "All tests passed!"
}

# Update version in code
update_version() {
    version="$1"
    version_num="${version#v}"
    
    log_info "Updating version to $version_num..."
    
    # Update cmd/version.go
    sed -i "s/Version = \"[^\"]*\"/Version = \"$version_num\"/" cmd/version.go
    
    # Commit the change
    git add cmd/version.go
    git commit -m "chore: bump version to $version"
}

# Create and push tag
create_tag() {
    version="$1"
    
    log_info "Creating tag $version..."
    git tag -a "$version" -m "Release $version"
    
    log_info "Pushing to remote..."
    git push origin main
    git push origin "$version"
    
    log_success "Tag $version created and pushed!"
}

# Main function
main() {
    version=$(get_version "${1:-}")
    validate_version "$version"
    
    log_info "Releasing unicd $version"
    
    check_branch
    check_clean
    run_tests
    update_version "$version"
    create_tag "$version"
    
    log_success "Release $version completed!"
    log_info "GitHub Actions will build and publish the release."
}

main "$@"
