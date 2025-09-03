#!/bin/bash

# Version management script for Klayengo
# Usage: ./scripts/version.sh <command> [args...]

set -e

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Function to get current version from git tags
get_current_version() {
    cd "$PROJECT_ROOT"
    git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"
}

# Function to validate version format
validate_version() {
    local version=$1
    if [[ ! $version =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        print_error "Invalid version format. Use vMAJOR.MINOR.PATCH (e.g., v1.2.3)"
        exit 1
    fi
}

# Function to check if required tools are available
check_dependencies() {
    local missing_tools=()

    if ! command -v git &> /dev/null; then
        missing_tools+=("git")
    fi

    if ! command -v sed &> /dev/null; then
        missing_tools+=("sed")
    fi

    if [[ ${#missing_tools[@]} -ne 0 ]]; then
        print_error "Missing required tools: ${missing_tools[*]}"
        exit 1
    fi
}

# Function to perform dry run
dry_run() {
    local new_version=$1

    validate_version "$new_version"

    print_info "DRY RUN - Would perform the following actions:"
    echo "  📝 Update version.go: Version = \"$new_version\""
    echo "  📝 Update README.md: **Version**: $new_version"
    echo "  📝 Update CHANGELOG.md: Add entry for $new_version"
    echo "  🔀 Git add and commit: \"Bump version to $new_version\""
    echo "  🏷️  Create git tag: $new_version"
    echo "  📤 Push to remote: main branch and $new_version tag"
    print_warning "This was a dry run. No changes were made."
}

# Function to create a new version
create_version() {
    local new_version=$1

    validate_version "$new_version"
    check_clean_workspace

    local current_version=$(get_current_version)

    print_info "Current version: $current_version"
    print_info "New version: $new_version"

    # Update version.go if it exists
    if [[ -f "$PROJECT_ROOT/version.go" ]]; then
        sed -i.bak "s/Version = \".*\"/Version = \"$new_version\"/" "$PROJECT_ROOT/version.go"
        rm "$PROJECT_ROOT/version.go.bak"
        print_success "Updated version.go"
    fi

    # Update CHANGELOG.md if it exists
    if [[ -f "$PROJECT_ROOT/CHANGELOG.md" ]]; then
        # Add new version entry to changelog
        local today=$(date +"%Y-%m-%d")
        local changelog_entry="## [$new_version] - $today\n\n### Added\n- Release $new_version\n\n### Changed\n- Version bump to $new_version\n"

        # Insert after the "Unreleased" section or at the top if no unreleased
        if grep -q "## \[Unreleased\]" "$PROJECT_ROOT/CHANGELOG.md"; then
            sed -i.bak "/## \[Unreleased\]/a \\\n$changelog_entry" "$PROJECT_ROOT/CHANGELOG.md"
        else
            sed -i.bak "1a $changelog_entry" "$PROJECT_ROOT/CHANGELOG.md"
        fi
        rm "$PROJECT_ROOT/CHANGELOG.md.bak"
        print_success "Updated CHANGELOG.md"
    fi

    # Commit changes
    cd "$PROJECT_ROOT"
    git add .
    git commit -m "Bump version to $new_version" || true

    # Create and push tag
    git tag -a "$new_version" -m "Release $new_version"
    git push origin main
    git push origin "$new_version"

    print_success "Version $new_version created and pushed!"
    print_info "GitHub Actions will automatically create a release."
}

# Function to show current version info
show_version() {
    cd "$PROJECT_ROOT"
    local current_version=$(get_current_version)
    local git_commit=$(git rev-parse --short HEAD)
    local build_date=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    echo "Current Version Information:"
    echo "  Version: $current_version"
    echo "  Git Commit: $git_commit"
    echo "  Build Date: $build_date"
    echo ""
    echo "Latest Tags:"
    git tag --sort=-version:refname | head -5
}

# Function to bump version
bump_version() {
    local bump_type=$1
    local current_version=$(get_current_version)

    # Remove 'v' prefix for calculation
    local version_numbers=$(echo "$current_version" | sed 's/^v//')
    IFS='.' read -r major minor patch <<< "$version_numbers"

    case $bump_type in
        major)
            major=$((major + 1))
            minor=0
            patch=0
            ;;
        minor)
            minor=$((minor + 1))
            patch=0
            ;;
        patch)
            patch=$((patch + 1))
            ;;
        *)
            print_error "Invalid bump type. Use: major, minor, or patch"
            exit 1
            ;;
    esac

    local new_version="v$major.$minor.$patch"
    print_info "Bumping $bump_type version: $current_version -> $new_version"
    create_version "$new_version"
}

# Function to check if workspace is clean
check_clean_workspace() {
    cd "$PROJECT_ROOT"
    if [[ -n $(git status --porcelain) ]]; then
        print_error "Workspace is not clean. Please commit or stash your changes before creating a version."
        exit 1
    fi
}

# Main script initialization
check_dependencies

# Main command handling
case "${1:-help}" in
    create|c)
        if [[ -z "$2" ]]; then
            print_error "Please specify a version (e.g., v1.2.3)"
            exit 1
        fi
        if [[ "$3" == "--dry-run" ]]; then
            dry_run "$2"
        else
            create_version "$2"
        fi
        ;;
    bump|b)
        if [[ -z "$2" ]]; then
            print_error "Please specify bump type: major, minor, or patch"
            exit 1
        fi
        if [[ "$3" == "--dry-run" ]]; then
            # Calculate new version for dry run
            current_version=$(get_current_version)
            version_numbers=$(echo "$current_version" | sed 's/^v//')
            IFS='.' read -r major minor patch <<< "$version_numbers"

            case $2 in
                major)
                    major=$((major + 1))
                    minor=0
                    patch=0
                    ;;
                minor)
                    minor=$((minor + 1))
                    patch=0
                    ;;
                patch)
                    patch=$((patch + 1))
                    ;;
            esac

            new_version="v$major.$minor.$patch"
            dry_run "$new_version"
        else
            bump_version "$2"
        fi
        ;;
    show|s|current)
        show_version
        ;;
    help|h|--help)
        echo "Klayengo Version Management Script"
        echo ""
        echo "Usage: $0 <command> [args...] [--dry-run]"
        echo ""
        echo "Commands:"
        echo "  create <version>    Create a new version (e.g., v1.2.3)"
        echo "  bump <type>         Bump version (major, minor, patch)"
        echo "  show                Show current version information"
        echo "  help                Show this help message"
        echo ""
        echo "Options:"
        echo "  --dry-run           Show what would be done without making changes"
        echo ""
        echo "Examples:"
        echo "  $0 create v1.2.3"
        echo "  $0 create v1.2.3 --dry-run"
        echo "  $0 bump minor"
        echo "  $0 bump patch --dry-run"
        echo "  $0 show"
        ;;
    *)
        print_error "Unknown command: $1"
        echo "Run '$0 help' for usage information."
        exit 1
        ;;
esac
