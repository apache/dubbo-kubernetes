#!/usr/bin/env bash

set -e

TARGET_PATH='github.com/apache/dubbo-kubernetes/api/type/v1alpha3'

find . -name "*.go" -type f | while read -r file; do
  sed -i.bak \
    -e 's|v1alpha3[[:space:]]*"/api/type/v1alpha3"|v1alpha3 "'"$TARGET_PATH"'"|g' \
    "$file"

  sed -i.bak \
    -e 's|"/api/type/v1alpha3"|"'"$TARGET_PATH"'"|g' \
    "$file"

  rm -f "${file}.bak"
done

echo "Fixed Success：/api/type/v1alpha3 -> ${TARGET_PATH}"
