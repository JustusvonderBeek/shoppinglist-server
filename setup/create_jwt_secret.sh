#!/bin/bash

echo "Creating new JWT secret"

secret=$(tr -dc 'A-Za-z0-9!#$%&'\''()*+,-./:;<=>?@[\]^_`{|}~' </dev/urandom | head -c 32; echo)

echo "Secret: $secret"

valid=$(date -d "90 days" --iso-8601=seconds)
echo "Valid until: $valid"

echo """{
	\"Secret\": \"$secret\",
	\"ValidUntil\": \"$valid\"
}""" > ../resources/jwtSecret.json

echo "Secret created"