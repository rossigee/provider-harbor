# Release Guide

This document describes the release process, versioning strategy, and CI/CD pipeline for provider-harbor.

## Versioning Strategy

provider-harbor follows **Semantic Versioning** (SemVer) with the format: `vMAJOR.MINOR.PATCH`

### Version Components

- **MAJOR**: Incompatible API changes or breaking changes to resource schemas
- **MINOR**: New features or resource types added (backward compatible)
- **PATCH**: Bug fixes and maintenance updates (backward compatible)

### Pre-release Versions

For development and testing:
- `v0.14.0-alpha.1` — Alpha release with limited testing
- `v0.14.0-beta.1` — Beta release ready for wider testing
- `v0.14.0-rc.1` — Release candidate, production-like

### Version Stability

| Version Range | Status | Support |
|--------------|--------|---------|
| v1.0.0+ | Stable | Full support, backwards compatible |
| v0.14.0+ | Production Ready | Full support, backwards compatible |
| v0.13.0-v0.13.x | Legacy | Security updates only |
| <v0.13.0 | Unsupported | No support |

## Release Timeline

### Development Cycle (2 weeks)

```
Week 1:
  Mon-Wed: Feature development and testing
  Thu-Fri: Bug fixes and stabilization

Week 2:
  Mon-Tue: Final validation and documentation
  Wed-Fri: Release preparation and testing
  Fri EOD: Tag and publish release
```

### Release Checklist

Before each release:

- [ ] All tests passing (65+ tests)
- [ ] Code reviewed and merged
- [ ] Documentation updated (README, examples, guides)
- [ ] CHANGELOG.md written with feature highlights
- [ ] Version numbers updated (VERSION file, API docs)
- [ ] Breaking changes documented if any
- [ ] Helm chart version bumped
- [ ] Container image validated locally

## CI/CD Pipeline

### Automated Quality Checks

**On every push to master or PR:**

1. **Lint** (golangci-lint 2.12.2)
   - Go fmt, goimports, vet, etc.
   - Custom linter rules
   - Timeout: 5 minutes

2. **Test** (Go 1.26.3)
   - Unit tests (55+)
   - Integration tests (5+)
   - Coverage report (target: ≥80%)
   - Timeout: 10 minutes

3. **Build**
   - Go binary build
   - Container image build
   - Cross-platform validation
   - Timeout: 10 minutes

4. **Security**
   - Trivy image scanning
   - Dependency vulnerability scanning
   - SAST analysis
   - Timeout: 5 minutes

### Release Pipeline

**On git tag push (e.g., `git tag v0.14.0`):**

1. **Verify Tag Format**
   - Must match `v*.*.*` pattern
   - Validates SemVer compliance

2. **Build & Test**
   - Full test suite
   - Image build

3. **Publish Artifacts**
   - Container image: `ghcr.io/rossigee/provider-harbor:v0.14.0`
   - Checksums and signatures
   - Release notes

4. **Create Release**
   - GitHub release with notes
   - Link to container image
   - Upgrade instructions

## Making a Release

### Step 1: Prepare Code

```bash
# Ensure clean working directory
git status

# Update version in VERSION file
echo "v0.14.0" > VERSION

# Update Helm chart if applicable
# Edit: chart/Chart.yaml
#   version: 0.14.0
#   appVersion: v0.14.0
```

### Step 2: Write Changelog

Create `CHANGELOG.md` entry:

```markdown
## v0.14.0 (2025-06-13)

### Features
- Added 12 resource types covering 60% Harbor API
- Implemented exponential backoff retry logic
- Added comprehensive production safety documentation

### Improvements
- Increased test coverage to 65+ tests
- Enhanced error messages and status conditions
- Optimized Harbor client connection pooling

### Fixes
- Fixed scan controller Update method behavior
- Corrected artifact vulnerability field names
- Resolved test file visibility issues

### Security
- Added CEL validation rules documentation
- Enhanced credential handling in examples
- Implemented security hardening guide

### Breaking Changes
None in this release (backward compatible from v0.13.0)

### Upgrade Notes
- No action required for existing deployments
- New resources can be used immediately
- All existing resources continue to work unchanged

### Contributors
- @rossigee - Core implementation
- Community - Testing and feedback
```

### Step 3: Commit and Tag

```bash
# Commit version and changelog
git add VERSION CHANGELOG.md
git commit -m "chore: Release v0.14.0

- 12 resource types with 60% API coverage
- 65+ comprehensive tests
- Production safety and readiness documentation
- Enhanced error handling and retry logic"

# Create annotated tag
git tag -a v0.14.0 -m "Release v0.14.0

Features:
- Complete Harbor project, repository, artifact management
- CI/CD robot accounts with expiration
- Webhook automation for scan notifications
- Cross-registry replication and retention policies
- Scanner registration and user management

See CHANGELOG.md for full details"

# Push to GitHub
git push origin master
git push origin v0.14.0
```

### Step 4: GitHub Actions

The release workflow automatically:

1. Validates tag format
2. Runs full test suite
3. Builds container image
4. Publishes to GHCR
5. Creates GitHub release
6. Generates checksums

Monitor the workflow in `.github/workflows/release.yml`

### Step 5: Verify Release

```bash
# Verify container image exists
docker pull ghcr.io/rossigee/provider-harbor:v0.14.0

# List available tags
docker image ls ghcr.io/rossigee/provider-harbor

# Check GitHub release
gh release view v0.14.0

# Verify tag in git
git tag -l | grep v0.14.0
```

## Version-Specific Documentation

### Upgrading from v0.13.0 to v0.14.0

**No breaking changes!** All existing resources continue to work.

```bash
# Update provider image in your deployment
kubectl set image deployment/provider-harbor \
  provider=ghcr.io/rossigee/provider-harbor:v0.14.0 \
  -n crossplane-system

# Verify rollout
kubectl rollout status deployment/provider-harbor \
  -n crossplane-system
```

### Managing Multiple Versions

For testing version upgrades:

```bash
# Deploy specific version
helm install provider-harbor ./chart \
  --set image.tag=v0.14.0

# Run side-by-side (different namespace)
helm install provider-harbor-test ./chart \
  --set image.tag=v0.14.0 \
  --namespace crossplane-system-test

# Compare behavior
kubectl get projects -n default
kubectl get projects -n default-test
```

## Release Cadence

### Regular Releases

- **Minor (Feature)**: Every 2-4 weeks
- **Patch (Bugfix)**: As needed
- **Major**: On significant API evolution

### Security Releases

Critical security issues trigger immediate patch release:

1. Assess severity
2. Develop fix
3. Tag as `v0.13.1` (or appropriate patch)
4. Mark as security release in GitHub
5. Publish advisory

### LTS (Long-Term Support)

Current LTS: v0.13.0
- Security updates: 12 months
- Bug fix updates: 6 months
- End of life: 2026-06-13

## Container Images

### Image Tags

Each release creates multiple image tags:

```bash
# Latest release
ghcr.io/rossigee/provider-harbor:latest
ghcr.io/rossigee/provider-harbor:v0.14.0

# Development
ghcr.io/rossigee/provider-harbor:dev
ghcr.io/rossigee/provider-harbor:master

# Latest in major version
ghcr.io/rossigee/provider-harbor:v0
```

### Image Contents

- **Base**: ubuntu:24.04
- **Provider Binary**: Statically linked Go binary
- **Health Checks**: Readiness and liveness probes
- **Non-root User**: Runs as unprivileged account
- **Size**: ~150MB

## Helm Chart Releases

Helm chart versioning matches provider versioning:

```yaml
# chart/Chart.yaml
apiVersion: v2
name: provider-harbor
version: 0.14.0
appVersion: v0.14.0
```

### Chart Distribution

Charts published to OCI registry:

```bash
# Install from OCI
helm install provider-harbor \
  oci://ghcr.io/rossigee/charts/provider-harbor \
  --version 0.14.0
```

## Publishing to Multiple Registries

### GHCR (Primary)

```bash
ghcr.io/rossigee/provider-harbor:v0.14.0
```

GitHub Actions handles publishing automatically.

### Docker Hub (Optional)

To publish to Docker Hub:

```bash
# Build and tag
docker build -t \
  rossigee/provider-harbor:v0.14.0 \
  -f cluster/images/xpkg.Dockerfile .

# Push
docker push rossigee/provider-harbor:v0.14.0
```

### Quay.io (Optional)

```bash
docker tag ghcr.io/rossigee/provider-harbor:v0.14.0 \
  quay.io/rossigee/provider-harbor:v0.14.0

docker push quay.io/rossigee/provider-harbor:v0.14.0
```

## Release Artifacts

Each release generates:

- ✅ Container image (GHCR)
- ✅ Binary artifacts (checksums, signatures)
- ✅ Helm chart
- ✅ Release notes
- ✅ API documentation
- ✅ Migration guide (if applicable)

## Support Policy

### Supported Versions

| Version | Released | Support Until | Status |
|---------|----------|---------------|--------|
| v0.14.0 | 2025-06-13 | 2025-12-13 | Current |
| v0.13.0 | 2025-01-15 | 2026-01-15 | LTS |
| v0.12.x | 2024-09-XX | 2024-12-XX | EOL |

### Reporting Issues

Found an issue? Please report:

1. **GitHub Issues**: Feature requests, bugs, documentation
2. **Security Issues**: Contact maintainer privately
3. **Questions**: GitHub Discussions

## Post-Release Tasks

After publishing a release:

- [ ] Announce on Slack/community channels
- [ ] Update documentation links
- [ ] Monitor for issues and feedback
- [ ] Update downstream projects
- [ ] Archive release notes
- [ ] Plan next release cycle

## Troubleshooting Releases

### Tag Already Exists

```bash
# Force delete local tag
git tag -d v0.14.0

# Force delete remote tag
git push origin :refs/tags/v0.14.0

# Create new tag
git tag -a v0.14.0 -m "Release v0.14.0"
git push origin v0.14.0
```

### Workflow Failure

Check GitHub Actions logs:

```bash
# View workflow runs
gh run list

# View specific run logs
gh run view <RUN_ID> --log

# Re-run failed job
gh run rerun <RUN_ID>
```

### Image Push Failures

```bash
# Verify GHCR credentials
echo $CR_PAT | docker login ghcr.io \
  -u <USERNAME> \
  --password-stdin

# Verify permissions
gh auth status

# Manually push image
docker build -t ghcr.io/rossigee/provider-harbor:v0.14.0 .
docker push ghcr.io/rossigee/provider-harbor:v0.14.0
```

## Questions?

For questions about the release process:

1. Check this guide
2. Review existing releases on GitHub
3. Open a GitHub discussion
4. Contact maintainer
