CREATE DATABASE IF NOT EXISTS shoppinglist;

use shoppinglist;

DROP USER IF EXISTS 'cloudsheeptech'@'localhost';
CREATE USER IF NOT EXISTS 'cloudsheeptech'@'localhost' IDENTIFIED BY '<password>';

SELECT User FROM mysql.user;

GRANT ALL PRIVILEGES ON shoppinglist.* TO 'cloudsheeptech'@'localhost' WITH GRANT OPTION;

FLUSH PRIVILEGES;

SHOW GRANTS FOR 'cloudsheeptech'@'localhost';

-- Start creating the database tables

DROP TABLE IF EXISTS shoppers;
DROP TABLE IF EXISTS items;
DROP TABLE IF EXISTS shoppinglists;

CREATE TABLE shoppers (
    id          INT AUTO_INCREMENT NOT NULL,
    name        VARCHAR(256)    NOT NULL,
    favRecipe   INT,
    PRIMARY KEY (`id`)
);

CREATE TABLE items (
    id      INT AUTO_INCREMENT NOT NULL,
    name    VARCHAR(256)    NOT NULL,
    image   VARCHAR(256)    NOT NULL,
    PRIMARY KEY (`id`)
);

CREATE TABLE shoppinglists (
    id      INT AUTO_INCREMENT NOT NULL,
    listId  INT NOT NULL,
    itemId  INT NOT NULL,
    quantity INT NOT NULL,
    PRIMARY KEY (`id`)
);