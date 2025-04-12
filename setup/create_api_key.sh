#!/bin/bash

function generateJWT() {
  payload="$1"
#  echo "Payload: $payload"
  destination="$2"
#  echo "Destination: $destination"
  signingSecretFile="$3"

  # Construct the header
  jwt_header=$(echo -n '{"alg":"HS256","typ":"JWT"}' | base64 -w 0 | sed s/\+/-/g | sed 's/\//_/g' | sed -E s/=+$//)

  # Read the secret from the jwtSecret.json file create by the other script
  if [ ! -f "$signingSecretFile" ]; then
      echo "JWT secret file not found! Create the jwtSecret first by running 'create_jwt_secret.sh' first."
      exit 1
  fi
  secret=$(cat "$signingSecretFile" | jq -r .Secret)

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

outputPath="$1"
if [[ $# -lt 1 ]]; then
  outputPath="./"
fi

secretFile="${outputPath}apiKey.secret"
jwtFile="${outputPath}apiKey.jwt"
signingSecretFile="${outputPath}/jwtSecret.json"
randomData=$(openssl rand -base64 32)
validUntil=$(date -d "90 days" --iso-8601=seconds)
echo "{\"secret\":\"$randomData\",\"validUntil\":\"$validUntil\"}" > "${secretFile}"

privateClaims=$(echo -n "{\"key\":\"$randomData\",\"validUntil\":\"$validUntil\",\"admin\":true}" )
#echo "$privateClaims"
base64PrivateClaims=$(echo "$privateClaims" | base64 -w 0 | sed 's/\+/-/g' | sed 's/\//_/g' |  sed -E 's/=+$//' )
#echo "Base64: '$base64PrivateClaims'"

# Convert the payload into a JWT token
generateJWT "$base64PrivateClaims" "$jwtFile" "$signingSecretFile"

echo "Secret stored into '$secretFile'"
echo "API key successfully created"