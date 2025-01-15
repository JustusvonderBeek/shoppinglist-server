#!/bin/bash

echo "Exporting database users..."

dbName="shoppinglist"
userTable="shoppers"

passwordFile="../resources/db.json"
exportFile="./users.csv"

dbUser=$(cat "$passwordFile" | jq -r .DBUser)
dbPassword=$(cat "$passwordFile" | jq -r .DBPwd)

query="SELECT * INTO OUTFILE '$exportFile' FIELDS TERMINATED BY ',' OPTIONALLY ENCLOSED BY '\"' LINES TERMINATED BY '\n' FROM $userTable;"
output=$(mysql -u "$dbUser" --password="$dbPassword" -D "$dbName" -e "$query")
#output=$(mysql -D "$dbName" -e "$query")
#echo "$output"

queryWithoutFile="SELECT CONCAT(id, ',\"', username, '\",\"', passwd, '\",\"', role, '\",\"', created, '\",\"', lastLogin, '\"') FROM $userTable"
mysql -u "$dbUser" --password="$dbPassword" -D "$dbName" --skip-column-names -e "$queryWithoutFile" > "$exportFile"

#mv "/var/lib/mysql/$exportFile" "./$exportFile"

echo "Exported users!"