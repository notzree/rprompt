# Homebrew Tap for rprompt

This directory contains the Homebrew formula for installing rprompt.

## Installation

You can install rprompt using Homebrew:

```bash
brew tap notzree/rprompt
brew install rprompt
```

## Development

To update the formula:

1. Update the version number in `Formula/rprompt.rb`
2. Update the SHA256 hashes for the new release binaries
3. Test the formula locally:
   ```bash
   brew install --build-from-source Formula/rprompt.rb
   ``` 