# Homebrew Formula for kportforward

This directory contains the Homebrew formula for installing kportforward on macOS.

## Usage

Users can install kportforward using:

```bash
brew tap catio-tech/kportforward https://github.com/catio-tech/kportforward
brew install kportforward
```

## Updating the Formula

When releasing a new version:

1. Update the `version` field in `kportforward.rb`
2. The SHA256 checksums are automatically calculated by Homebrew on first install

## Maintenance

- Keep this formula updated with each new release
- Test installation on both Intel and ARM Macs
- Ensure the formula works correctly with `brew audit --strict kportforward`