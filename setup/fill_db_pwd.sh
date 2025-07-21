#!/bin/bash

echo "This script requires jq to parse JSON files and SED to replace strings in the SQL file"
echo "Filling the database SQL file with the correct credentials"

database_file="create_mysql_db.sql"
replaced_database_file="filled_database.sql"
replaced_docker_db_setup_file="../compose-db-setup/filled_database.sql"
database_information="../resources/config.json"
placeholder_username="<username>"
placeholder_locality="<locality>"
placeholder_password="<password>"

# Replace each string with the given information from the 'db_conf.json' file
username=$(cat "$database_information" | jq -r .Database.User)
password=$(cat "$database_information" | jq -r .Database.Password)
dbaddress=$(cat "$database_information" | jq -r .Database.Host)
locality=${dbaddress%%:*}
# echo "$username, $password, $locality"

# '-i' inline; '-e' expression; 's/' search?; '/g' globally; 's/searchString/replaceString/g' replace globally
replaced_username=$(sed -e "s/$placeholder_username/$username/g" "$database_file")
replaced_locality=$(sed -e "s/$placeholder_locality/$locality/g" <<< $replaced_username)
# Can make problems if the password contains the same character as the delimiter of sed!
replaced_password=$(sed -e "s|$placeholder_password|$password|g" <<< $replaced_locality)

echo "-- THIS FILE WAS AUTOMATICALLY GENERATED!" > "$replaced_database_file"
echo "-- DO NOT MANUALLY MODIFY THIS FILE" >> "$replaced_database_file"
echo "" >> "$replaced_database_file"
echo "$replaced_password" >> "$replaced_database_file"

# Write the filled file to the compose setup folder as well
echo "$replaced_password" > "$replaced_docker_db_setup_file"

echo "Done"