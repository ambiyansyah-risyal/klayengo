#!/bin/bash

# Script to install git hooks
# Run this script to copy hooks from the githooks directory to .git/hooks

HOOKS_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )/githooks"

if [ ! -d "$HOOKS_DIR" ]; then
    echo "Error: githooks directory not found at $HOOKS_DIR"
    exit 1
fi

GIT_HOOKS_DIR="$( git rev-parse --git-dir )/hooks"

if [ ! -d "$GIT_HOOKS_DIR" ]; then
    echo "Error: .git/hooks directory not found"
    exit 1
fi

# Copy all hook files from githooks directory to .git/hooks
for hook in "$HOOKS_DIR"/*; do
    hook_name=$(basename "$hook")
    destination="$GIT_HOOKS_DIR/$hook_name"
    
    cp "$hook" "$destination"
    chmod +x "$destination"
    
    echo "Installed $hook_name to $GIT_HOOKS_DIR"
done

echo "Git hooks installed successfully!"