#!/bin/bash

# Execute this script to make the system ready to run the server

echo "Creating TLS Certificates and setup database"

# Create TLS Certificates
./create_tls_cert.sh

echo "Is the password for the database already correctly inserted into the SQL file?"

read -p "Press enter to continue"

# Create the tables in the database
sudo mysql < ./create_mysql_db.sql

# Create the JWT secret
./create_jwt_secret.sh

echo "Setup done"