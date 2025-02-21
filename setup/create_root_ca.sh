#!/bin/bash

echo "Creating new RootCA"

rootConfig="root-ca.conf"
rootCert="root_ca_cert.pem"

openssl req -x509 -config "$rootConfig" -days 365 -newkey rsa:4096 -sha256 -out "$rootCert" -outform PEM

echo "Done"