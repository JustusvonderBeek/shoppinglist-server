#!/bin/bash

echo "Importing database users..."

importFile="./users.csv"
passwordFile="../resources/db.json"

dbName="shoppinglist"
userTable="shoppers"
dbUser=$(cat "$passwordFile" | jq -r .DBUser)
dbPassword=$(cat "$passwordFile" | jq -r .DBPwd)

query="LOAD DATA INFILE '$importFile' INTO TABLE $userTable FIELDS TERMINATED BY ',' OPTIONALLY ENCLOSED BY '\"' LINES TERMINATED BY '\n' (id, username, passwd, created, lastLogin) SET role = 'user';"
echo "$query"
mysql -u "$dbUser" --password="$dbPassword" -D "$dbName" -e "$query"

echo "Database users imported!"