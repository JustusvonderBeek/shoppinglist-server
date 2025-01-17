#!/bin/bash

# Execute this script to make the system ready to run the server

echo "Creating TLS Certificates and setup database"

outputDirectory="../resources/"

# Create TLS Certificates
./create_tls_cert.sh "$outputDirectory"

echo "Is the password for the database already correctly inserted into the SQL file?"

read -p "Press enter to continue"

# Create the tables in the database
sudo mysql < ./create_mysql_db.sql

# Create the JWT secret
./create_jwt_secret.sh "$outputDirectory"

# Create the API key
./create_api_key.sh "$outputDirectory"

# Create the admin user
./fill_admin_pwd.sh "$outputDirectory"
./create_admin_user_pwd.sh

echo "Setup done"