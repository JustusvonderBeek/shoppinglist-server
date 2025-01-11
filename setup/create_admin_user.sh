#!/bin/bash

echo "Creating new admin user..."

username="<username>"
password="<password>"
requestContent="{ \"onlineId\": 0, \"username\": \"$username\", \"password\": \"$password\" }"
apiKey="<api-key>"
requestUrl="https://localhost:46152/v1/users"

curl -k -X POST -H "Content-Type: application/json" -H "x-api-key: $apiKey" -d "$requestContent" "$requestUrl"

echo ""
echo "Created new admin user"