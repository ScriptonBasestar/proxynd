#!/bin/bash
cd /Users/archmagece/myopen/scripton/proxynd
golangci-lint run ./... --skip-files "examples/.*" 2>&1
