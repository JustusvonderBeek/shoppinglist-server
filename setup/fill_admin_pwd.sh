#!/bin/bash

echo "Filling password information into 'create_admin_user.sh'"

script_file="create_admin_user.sh"
replaced_script_file="create_admin_user_pwd.sh"
api_key_file="../resources/apiKey.jwt"
database_information="../resources/admin.json"
placeholder_username="<username>"
placeholder_password="<password>"
placeholder_api_key="<api-key>"

# Replace each string with the given information from the 'db.json' file
username=$(cat "$database_information" | jq -r .username)
password=$(cat "$database_information" | jq -r .password)
apiKey=$(cat "$api_key_file")

replaced_username=$(sed -e "s/$placeholder_username/$username/g" "$script_file")
# Can make problems if the password contains the same character as the delimiter of sed!
replaced_password=$(sed -e "s|$placeholder_password|$password|g" <<< "$replaced_username")
replaced_api_key=$(sed -e "s/$placeholder_api_key/$apiKey/g" <<< "$replaced_password")

echo "$replaced_api_key" > "$replaced_script_file"

echo "Password informationed filled"