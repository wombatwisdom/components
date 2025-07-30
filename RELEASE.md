# Release Process

This document describes the release process for WombatWisdom Components.

## Overview

WombatWisdom Components uses a multi-module Go workspace structure. Each component can be versioned independently, but we also maintain synchronized releases across all components for major milestones.

## Versioning Strategy

We follow [Semantic Versioning](https://semver.org/):
- **MAJOR** version (v1.0.0) - Incompatible API changes
- **MINOR** version (v0.1.0) - New functionality, backwards compatible
- **PATCH** version (v0.0.1) - Backwards compatible bug fixes
- **Release Candidates** (v0.1.0-rc1) - Pre-release versions for testing

## Tag Structure

### Main Repository Tag
- Format: `vX.Y.Z` or `vX.Y.Z-rcN`
- Example: `v0.1.0-rc2`, `v1.0.0`

### Component Tags
- Format: `components/{name}/vX.Y.Z`
- Examples:
  - `components/nats/v0.1.0-rc2`
  - `components/aws-s3/v0.1.0-rc2`
  - `framework/v0.1.0-rc2`

## Release Types

### 1. Release Candidate (RC)
Pre-release versions for testing and validation.

```bash
# Create next RC version
task release:rc

# Create specific RC version
task release:rc VERSION=v0.2.0-rc1
```

### 2. Final Release
Stable release versions.

```bash
# Create release from current RC
task release:final

# Create specific version release
task release:final VERSION=v1.0.0
```

### 3. Component-Only Release
Release a specific component independently.

```bash
# Release specific component
task release:component COMPONENT=nats VERSION=v0.1.1
```

## Release Process

### Prerequisites
1. Ensure all tests pass: `task test`
2. Ensure code is properly formatted: `task format`
3. Ensure CI is green on main branch
4. Update CHANGELOG.md with release notes

### Available Release Tasks

```bash
# Show current version
task release:current

# List release history
task release:list

# Generate changelog
task release:changelog                    # From current version to HEAD
task release:changelog FROM=v0.1.0 TO=v0.2.0  # Between specific versions

# Verify all tags exist for a version
task release:verify VERSION=v0.1.0

# Create release candidate
task release:rc                          # Auto-increment RC version
task release:rc VERSION=v0.2.0-rc1      # Specific RC version

# Create final release
task release:final                       # Promote latest RC to final
task release:final VERSION=v0.2.0       # Specific version

# Create GitHub release
task release:github VERSION=v0.1.0
task release:github VERSION=v0.1.0-rc1 PRERELEASE=true

# Release individual component
task release:component COMPONENT=nats VERSION=v0.1.1
```

### Step-by-Step Release

#### 1. Review Current State
```bash
# Check current version
task release:current

# Generate changelog to review changes
task release:changelog
```

#### 2. Create Release Candidate
```bash
# Automatically determine next RC version
task release:rc

# Or specify version
task release:rc VERSION=v0.2.0-rc1
```

This will:
- Run pre-release checks (clean working directory, on main branch, up to date)
- Create annotated tags for main repo and all components
- Push tags to GitHub
- Trigger CI/CD workflows

#### 3. Verify Release
```bash
# Verify all tags were created
task release:verify VERSION=v0.2.0-rc1
```

#### 4. Test Release Candidate
- Deploy RC to staging environment
- Run integration tests
- Gather feedback from users

#### 5. Create Final Release
```bash
# Promote current RC to final release
task release:final

# Or create specific version
task release:final VERSION=v0.2.0
```

#### 6. Create GitHub Release
```bash
# Create GitHub release
task release:github VERSION=v0.2.0

# For release candidates
task release:github VERSION=v0.2.0-rc1 PRERELEASE=true
```

#### 7. Post-Release Tasks
- Update documentation
- Announce release
- Update dependent projects

## Manual Release Process

If you need to manually create tags:

```bash
# 1. Create main repository tag
git tag -a v0.1.0-rc2 -m "Release candidate 2"

# 2. Create component tags
git tag -a components/aws-eventbridge/v0.1.0-rc2 -m "AWS EventBridge component release candidate 2"
git tag -a components/aws-s3/v0.1.0-rc2 -m "AWS S3 component release candidate 2"
git tag -a components/ibm-mq/v0.1.0-rc2 -m "IBM MQ component release candidate 2"
git tag -a components/mqtt/v0.1.0-rc2 -m "MQTT component release candidate 2"
git tag -a components/nats/v0.1.0-rc2 -m "NATS component release candidate 2"
git tag -a framework/v0.1.0-rc2 -m "Framework release candidate 2"

# 3. Push all tags
git push origin --tags
```

## GitHub Release Creation

After tagging, create a GitHub release:

```bash
# Create release from tag
task release:github VERSION=v0.1.0

# Create pre-release from RC
task release:github VERSION=v0.1.0-rc2 PRERELEASE=true
```

## Component Independence

While we typically release all components together, individual components can be released independently:

```bash
# Release only NATS component
task release:component COMPONENT=nats VERSION=v0.1.1

# Release only framework
task release:component COMPONENT=framework VERSION=v0.1.1
```

## Rollback Process

If issues are discovered after release:

1. **Do not delete tags** - Tags are immutable once pushed
2. Create a patch release with fixes
3. Document known issues in GitHub Release notes
4. Communicate with users about the issue

## CI/CD Integration

The `.github/workflows/release.yml` workflow automatically:
- Builds all components when tags are pushed
- Runs tests
- Creates GitHub releases with artifacts
- Publishes to package registries (if configured)

## Best Practices

1. **Always test RCs thoroughly** before promoting to final release
2. **Use conventional commits** for automatic changelog generation
3. **Document breaking changes** clearly in CHANGELOG.md
4. **Coordinate releases** across team members
5. **Tag from main branch** after all PRs are merged
6. **Never force-push tags** - create new versions instead

## Troubleshooting

### Tag already exists
```bash
# Check existing tags
git tag -l "v0.1.0*"

# Create next version instead
task release:rc VERSION=v0.1.0-rc3
```

### Push rejected
```bash
# Ensure you have latest changes
git pull --tags

# Try pushing again
git push origin --tags
```

### Missing component tags
```bash
# List all tags for a component
git tag -l "components/nats/*"

# Create missing tag manually
git tag -a components/nats/v0.1.0 -m "NATS component v0.1.0"
git push origin components/nats/v0.1.0
```