#!/bin/bash

echo "Creating new JWT secret"

tr -dc 'A-Za-z0-9!"#$%&'\''()*+,-./:;<=>?@[\]^_`{|}~' </dev/urandom | head -c 32; echo

echo "Secret created"