# Release Process

This document describes the release process for WombatWisdom Components.

## Overview

WombatWisdom Components uses [goreleaser](https://goreleaser.com/) to automate the release process. All components are released together with a single version number.

## Versioning Strategy

We follow [Semantic Versioning](https://semver.org/):
- **MAJOR** version (v1.0.0) - Incompatible API changes
- **MINOR** version (v0.1.0) - New functionality, backwards compatible
- **PATCH** version (v0.0.1) - Backwards compatible bug fixes
- **Release Candidates** (v0.1.0-rc1) - Pre-release versions for testing

## Quick Start

### Create a Release Candidate
```bash
# Automatically determine next RC version
task release:rc

# Or specify a version
task release:rc VERSION=1.0.0-rc1
```

### Create a Final Release
```bash
# Promote current RC to final
task release:final

# Or specify a version
task release:final VERSION=1.0.0
```

### Create a Local Snapshot
```bash
# Test the release process locally without pushing
task release:snapshot
```

## Release Tasks

### Code Quality
- `task check` - Run all quality checks (lint, format, vet)
- `task test` - Run all tests
- `task test:unit` - Run unit tests only
- `task test:integration` - Run integration tests only
- `task test:race` - Run tests with race detector

### Release Management
- `task release:snapshot` - Create local snapshot (no git operations)
- `task release:rc` - Create release candidate
- `task release:final` - Create final release
- `task release:list` - List recent releases
- `task release:current` - Show current version
- `task release:validate VERSION=X.Y.Z` - Validate version format

## Release Process

### 1. Prepare for Release

Ensure your code is ready:
```bash
# Run quality checks
task check

# Run all tests
task test

# Check current version
task release:current
```

### 2. Create Release Candidate

Create a release candidate for testing:
```bash
# Auto-increment RC version
task release:rc
```

This will:
1. Run pre-release checks (clean working directory, on main branch)
2. Run code quality checks (`task check`)
3. Run all tests (`task test`)
4. Create and push a git tag (e.g., `v1.0.0-rc1`)
5. Trigger GitHub Actions to create the release using goreleaser

### 3. Test Release Candidate

- Deploy RC to staging environment
- Run integration tests
- Gather feedback from users

### 4. Create Final Release

Once the RC is validated:
```bash
# Promote RC to final release
task release:final
```

Or create multiple RCs if needed:
```bash
task release:rc  # Creates v1.0.0-rc2, rc3, etc.
```

### 5. Release Automation

Once a tag is pushed, GitHub Actions will:
1. Run goreleaser to create the GitHub release
2. Generate changelog from commit messages
3. Create source archives
4. Upload checksums

## Conventional Commits

For better changelogs, use conventional commit messages:
- `feat:` - New features
- `fix:` - Bug fixes
- `docs:` - Documentation changes
- `test:` - Test improvements
- `refactor:` - Code refactoring
- `chore:` - Maintenance tasks

Example:
```bash
git commit -m "feat: add support for AWS S3 multipart uploads"
git commit -m "fix: resolve race condition in MQTT client"
```

## Manual Release Process

If you need to create a release manually:

```bash
# Create and push tag
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# GitHub Actions will automatically create the release
```

## Rollback Process

If issues are discovered after release:

1. **Do not delete tags** - Tags are immutable once pushed
2. Create a patch release with fixes
3. Update release notes to document known issues
4. Communicate with users about the issue

## Troubleshooting

### Tag already exists
```bash
# Check existing tags
git tag -l "v1.0.0*"

# Create next version
task release:rc VERSION=1.0.1-rc1
```

### Push rejected
```bash
# Ensure you have latest changes
git pull --tags
git push origin --tags
```

### Release checks failing
```bash
# Check working directory is clean
git status

# Ensure on main branch
git checkout main
git pull origin main

# Run checks manually
task check
task test
```

## Best Practices

1. **Always test RCs thoroughly** before promoting to final release
2. **Use conventional commits** for automatic changelog generation
3. **Document breaking changes** clearly in commit messages
4. **Run `task check` and `task test`** before creating releases
5. **Tag from main branch** after all PRs are merged
6. **Never force-push tags** - create new versions instead

## Component Changes

While all components share the same version, the goreleaser changelog will show which components were modified in each release based on the changed files.