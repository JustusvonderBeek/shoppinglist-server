#!/bin/bash

echo "Creating new JWT secret"

secret=$(tr -dc 'A-Za-z0-9!"#$%&'\''()*+,-./:;<=>?@[\]^_`{|}~' </dev/urandom | head -c 32; echo)
echo "Secret: $secret"

echo """{
	\"Secret\": \"$secret\",
	\"ValidUntil\": \"2024-01-01T15:00:00+01:00\"
}""" > ../resources/jwtSecret.json

echo "Secret created"