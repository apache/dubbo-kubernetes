#!/bin/bash
set -e
cd "$(dirname "$0")"

# This script will be used when protoc is fixed
# For now, proto files are generated in Dockerfile
echo "Proto files are generated in Dockerfile during build"
echo "To generate locally, fix protoc first:"
echo "  brew reinstall protobuf abseil"
echo ""
echo "Then run:"
echo "  protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/echo.proto"
