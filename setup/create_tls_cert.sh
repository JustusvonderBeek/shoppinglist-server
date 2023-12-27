#!/bin/bash

outputPath="../resources/"
outputCert="shoppinglist.crt"
outputKey="shoppinglist.pem"

echo "Creating new certificate and keyfile."

# Creating the resource folder if it does not exists yet
mkdir -p $outputPath

openssl req -x509 -newkey rsa:4096 -keyout "$outputPath$outputKey" -out "$outputPath$outputCert" -sha512 -days 365 -nodes -subj "/C=DE/ST=Bavaria/L=Munich/O=Cloudsheeptech/OU=Shoppinglist/CN=shop.cloudsheeptech.com"

echo "Certificate and keyfile successfully created!"