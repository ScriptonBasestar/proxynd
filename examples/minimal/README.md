# Minimal Configuration Examples

This directory contains minimal configuration examples for getting started with ProxyND quickly.

## Files

### `config.minimal.yaml`
The absolute minimum configuration needed to run ProxyND. This configuration:
- Enables basic HTTP server on port 8080
- Sets up file-based caching
- Configures minimal logging

Perfect for local development or testing.

### `config.example.yaml`
A basic example that includes:
- Common proxy configurations
- Basic cache settings
- Standard logging configuration
- Health check endpoints

Good starting point for customization.

## Usage

```bash
# Copy to your config directory
cp config.minimal.yaml ../../config.yaml

# Set required environment variables
export CONFIG_DIR=./config
export STORAGE_DIR=./storage

# Run ProxyND
../../proxynd
```