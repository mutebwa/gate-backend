#!/bin/sh

# Decode base64 service account key if provided
if [ ! -z "$GOOGLE_APPLICATION_CREDENTIALS_BASE64" ]; then
    echo "Decoding service account credentials..."
    echo "$GOOGLE_APPLICATION_CREDENTIALS_BASE64" | base64 -d > /root/serviceAccountKey.json
    export FIREBASE_CREDENTIALS_PATH=/root/serviceAccountKey.json
fi

# Run the server
exec ./server
