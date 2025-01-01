#!/bin/bash

function generateJWT() {
  payload="$1"
#  echo "Payload: $payload"
  destination="$2"
#  echo "Destination: $destination"

  # Construct the header
  jwt_header=$(echo -n '{"alg":"HS256","typ":"JWT"}' | base64 -w 0 | sed s/\+/-/g | sed 's/\//_/g' | sed -E s/=+$//)

  # Read the secret from the jwtSecret.json file create by the other script
  if [ ! -f ../resources/jwtSecret.json ]; then
      echo "JWT secret file not found! Create the jwtSecret first by running 'create_jwt_secret.sh' first."
      exit 1
  fi
  secret=$(cat "../resources/jwtSecret.json" | jq -r .Secret)

  # Convert secret to hex (not base64)
  hexsecret=$(echo -n "$secret" | xxd -p | paste -sd "")

  # Calculate hmac signature -- note option to pass in the key as hex bytes
  hmac_signature=$(echo -n "${jwt_header}.${payload}" |  openssl dgst -sha256 -mac HMAC -macopt hexkey:$hexsecret -binary | base64 -w 0 | sed 's/\+/-/g' | sed 's/\//_/g' | sed -E 's/=+$//' )

  # Create the full token
  jwt="${jwt_header}.${payload}.${hmac_signature}"

  echo "$jwt" > "$destination"
  echo "Wrote API Key into '$destination'"
}

echo "Creating new API key..."

secretFile="../resources/apiKey.secret"
jwtFile="../resources/apiKey.jwt"
randomData=$(openssl rand -base64 32)
validUntil=$(date -d "90 days" --iso-8601=seconds)
echo "{\"secret\":\"$randomData\",\"validUntil\":\"$validUntil\"}" > "$secretFile"

privateClaims=$(echo -n "{\"key\":\"$randomData\",\"validUntil\":\"$validUntil\",\"userId\":\"admin\"}" )
#echo "$privateClaims"
base64PrivateClaims=$(echo "$privateClaims" | base64 -w 0 | sed 's/\+/-/g' | sed 's/\//_/g' |  sed -E 's/=+$//' )
#echo "Base64: '$base64PrivateClaims'"

# Convert the payload into a JWT token
generateJWT "$base64PrivateClaims" "$jwtFile"

echo "Secret stored into '$secretFile'"
echo "API key successfully created"