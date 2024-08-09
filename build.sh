#!/bin/bash

set -a
source .env
set +a

# to set environment variable run `export GOOGLE_CLIENT_ID=the_id_from_GCD`
go build -ldflags "\
-X 'github.com/CodedMasonry/cc-printer/providers/google.GoogleClientID=$GOOGLE_CLIENT_ID' \
-X 'github.com/CodedMasonry/cc-printer/providers/google.GoogleClientSecret=$GOOGLE_CLIENT_SECRET'\
"

echo "Build successful"