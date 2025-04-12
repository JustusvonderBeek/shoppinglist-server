#!/bin/bash

outputPath="$1"
if [[ $# -lt 1 ]]; then
  outputPath="./"
fi
outputCert="shoppinglist.crt"
outputKey="shoppinglist.pem"

serverSign="servercert.csr"
serverCert=""
serverConfig="server-cert.conf"
rootConfig="root-ca.conf"

echo "Creating new certificate and keyfile."

# Creating the resource folder if it does not exists yet
mkdir -p $outputPath

openssl req -config "$serverConfig" -newkey rsa:4096 -sha256 -nodes -out "$serverSign" -outform PEM

if [[ ! -f index.txt ]]; then
  touch index.txt
fi
if [[ ! -f serial.txt ]]; then
  echo '01' > serial.txt
fi

openssl ca -config "$rootConfig" -policy signing_policy -extensions signing_req -out servercert.pem -infiles "$serverSign"

#openssl req -x509 -newkey rsa:4096 -keyout "$outputPath$outputKey" -out "$outputPath$outputCert" -sha512 -days 365 -nodes -subj "/C=DE/ST=Bavaria/L=Munich/O=Cloudsheeptech/OU=Shoppinglist/CN=shop.cloudsheeptech.com"

echo "Certificate and keyfile successfully created!"