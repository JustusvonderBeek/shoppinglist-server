#!/bin/bash

# Execute this script to make the system ready to run the server

echo "Creating TLS Certificates and setup database"

outputDirectory="../resources/"

# Create TLS Certificates
./create_tls_cert.sh "$outputDirectory"

# Extract the password from the db.json file and create a config
echo "Is the password for the database already correctly inserted into the resources/db.json config file?"
read -p "Press enter to continue"

./fill_db_pwd.sh

# Create the tables in the database
sudo mysql < ./filled_database.sql

# Create the JWT secret
./create_jwt_secret.sh "$outputDirectory"

# Create the API key
./create_api_key.sh "$outputDirectory"

# Create the admin user
./fill_admin_pwd.sh "$outputDirectory"
sudo chmod 0700 ./create_admin_user_pwd.sh
./create_admin_user_pwd.sh

echo "Setup done"