# GitHub Secrets Setup Guide

To enable full test coverage reporting, you need to set up the following GitHub secrets:

## Required Secrets

### CODECOV_TOKEN

This token allows GitHub Actions to upload coverage reports to Codecov.

#### Steps to get the token:

1. Go to [Codecov](https://codecov.io/)
2. Sign in with your GitHub account
3. Add the repository: `stormkit-io/stormkit-io`
4. Navigate to Settings → General
5. Copy the "Repository Upload Token"

#### Add to GitHub:

1. Go to your repository: `https://github.com/stormkit-io/stormkit-io`
2. Navigate to Settings → Secrets and variables → Actions
3. Click "New repository secret"
4. Name: `CODECOV_TOKEN`
5. Value: Paste the token from Codecov
6. Click "Add secret"

## Verifying Setup

After adding the secret:

1. Create a new branch or push changes
2. Check the GitHub Actions workflow runs
3. Look for "Upload coverage to Codecov" step
4. Verify coverage appears on: `https://codecov.io/gh/stormkit-io/stormkit-io`

## Optional: Codecov Badge

The README already includes a Codecov badge. Once setup is complete, it will show:
- Current coverage percentage
- Coverage trends
- Links to detailed reports

Badge URL: `https://codecov.io/gh/stormkit-io/stormkit-io/branch/main/graph/badge.svg`

## Troubleshooting

### Token not working

- Verify the token is copied correctly (no extra spaces)
- Check that the repository name matches exactly
- Ensure the token hasn't expired

### Coverage not showing

- Check GitHub Actions logs for errors
- Verify the coverage file is generated (`coverage.out`)
- Ensure the workflow has proper permissions

### Need help?

Contact Codecov support or check their [documentation](https://docs.codecov.com/).
