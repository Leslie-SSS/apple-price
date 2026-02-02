#!/bin/bash
# Build script for ApplePrice
cd "$(dirname "$0")"
/usr/bin/docker compose build
