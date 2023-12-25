#!/bin/bash

# Execute this script to make the system ready to run the server

echo "Creating TLS Certificates and setup database"

# Create TLS Certificates
./create_tls_cert.sh

# Create the tables in the database
sudo mysql < ./create_mysql_db.sql

echo "Setup done"