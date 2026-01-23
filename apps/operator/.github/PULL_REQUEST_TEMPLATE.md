## Description

<!-- Provide a brief description of the changes in this PR -->

## Type of Change

<!-- Mark the relevant option with an "x" -->

- [ ] 🐛 Bug fix (non-breaking change which fixes an issue)
- [ ] ✨ New feature (non-breaking change which adds functionality)
- [ ] 💥 Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] 📝 Documentation update
- [ ] 🔧 Configuration change
- [ ] ♻️ Code refactoring
- [ ] ⚡ Performance improvement
- [ ] ✅ Test update

## Related Issues

<!-- Link to related issues using #issue_number -->

Closes #
Related to #

## Changes Made

<!-- Describe the changes in detail -->

- Change 1
- Change 2
- Change 3

## Testing

<!-- Describe the tests you ran and their results -->

### Unit Tests

- [ ] All existing tests pass
- [ ] Added new unit tests
- [ ] Coverage is maintained or improved

### E2E Tests

- [ ] All E2E tests pass
- [ ] Added new E2E tests (if applicable)

### Manual Testing

<!-- Describe manual testing performed -->

```bash
# Commands used for testing
kubectl apply -f ...
```

**Test Results:**

- ✅ Test case 1: Passed
- ✅ Test case 2: Passed

## Checklist

<!-- Mark completed items with an "x" -->

### Code Quality

- [ ] Code follows the project's style guidelines (`make lint` passes)
- [ ] Code is properly formatted (`make fmt` applied)
- [ ] Self-review of code performed
- [ ] Code is commented where necessary
- [ ] No debugging code or console logs left

### Testing

- [ ] Unit tests added/updated and passing
- [ ] E2E tests passing (if applicable)
- [ ] Manual testing performed
- [ ] Edge cases considered and tested

### Documentation

- [ ] Documentation updated (if needed)
- [ ] README updated (if needed)
- [ ] API documentation updated (if API changes)
- [ ] DEVELOPMENT.md updated (if workflow changes)
- [ ] Comments in code are clear and helpful

### Dependencies

- [ ] No new dependencies added
- [ ] OR: New dependencies are justified and documented
- [ ] `go.mod` and `go.sum` are updated

### Security

- [ ] No secrets or sensitive data in code
- [ ] Security best practices followed
- [ ] No security vulnerabilities introduced

### Generated Files

- [ ] Generated files updated (`make generate manifests`)
- [ ] CRD manifests regenerated (if API changes)

## Screenshots / Logs

<!-- If applicable, add screenshots or logs to help explain your changes -->

```
Paste relevant logs here
```

## Deployment Notes

<!-- Any special instructions for deploying this change? -->

- Deployment step 1
- Deployment step 2

## Breaking Changes

<!-- If this is a breaking change, describe the impact and migration path -->

### Impact

- Component affected 1
- Component affected 2

### Migration Guide

```bash
# Steps to migrate
```

## Additional Notes

<!-- Any additional information that reviewers should know -->

## Reviewer Checklist

<!-- For reviewers -->

- [ ] Code changes reviewed and approved
- [ ] Tests are adequate
- [ ] Documentation is sufficient
- [ ] No security concerns
- [ ] CI/CD pipeline passes
