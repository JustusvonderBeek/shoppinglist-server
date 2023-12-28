-- Execute this file with the following command:
-- Log into mysql: 
-- sudo mysql
-- source ./create_mysql_db.sql;
-- OR directly:
-- sudo mysql < ./create_mysql_db.sql

CREATE DATABASE IF NOT EXISTS shoppinglist;

use shoppinglist;

DROP USER IF EXISTS 'cloudsheeptech'@'localhost';
CREATE USER IF NOT EXISTS 'cloudsheeptech'@'localhost' IDENTIFIED BY '<password>';

SELECT User FROM mysql.user;

GRANT ALL PRIVILEGES ON shoppinglist.* TO 'cloudsheeptech'@'localhost' WITH GRANT OPTION;

FLUSH PRIVILEGES;

SHOW GRANTS FOR 'cloudsheeptech'@'localhost';

-- Start creating the database tables

DROP TABLE IF EXISTS shoppers;  -- The users of our system
DROP TABLE IF EXISTS items;     -- The items that can be shopped and shared
DROP TABLE IF EXISTS shoppinglist; -- Holding generic list information
DROP TABLE IF EXISTS sharedList; -- Which user can access which list
DROP TABLE IF EXISTS itemsPerList;     -- The mapping of items to lists
DROP TABLE IF EXISTS recipe; -- The description of recipes
DROP TABLE IF EXISTS itemsPerRecipe; -- The items that make a recipe
DROP TABLE IF EXISTS sharedRecipe; -- Which user can access which recipe
DROP TABLE IF EXISTS history; -- What has been bought in the past

-- Table for AUTHENTICATION + AUTHORIZATION (mapping what lists / items can be seen)

CREATE TABLE shoppers (
    id          BIGINT       NOT NULL,
    username    VARCHAR(256) NOT NULL,
    passwd      VARCHAR(512) NOT NULL,
    PRIMARY KEY (`id`)
);

-- Table for Items that can be shopped. Shared among all users

CREATE TABLE items (
    id      INT AUTO_INCREMENT  NOT NULL,
    name    VARCHAR(256)        NOT NULL,
    icon    VARCHAR(256)        NOT NULL,
    PRIMARY KEY (`id`)
);

-- Table holding the list information + the mapping of items to lists

CREATE TABLE shoppinglist(
    id          INT AUTO_INCREMENT  NOT NULL,
    name        VARCHAR(256)        NOT NULL,
    creatorId   BIGINT              NOT NULL,
    PRIMARY KEY (`id`)
);

CREATE TABLE sharedList(
    id              INT AUTO_INCREMENT  NOT NULL,
    listId          INT                 NOT NULL,
    sharedWithId    BIGINT              NOT NULL,
    PRIMARY KEY (`id`)
);

CREATE TABLE itemsPerList (
    id          INT AUTO_INCREMENT  NOT NULL,
    listId      INT                 NOT NULL,
    itemId      INT                 NOT NULL,
    quantity    INT                 NOT NULL,
    checked     BIT(1)              NOT NULL,
    addedBy     BIGINT              NOT NULL,
    PRIMARY KEY (`id`)
);

-- Table holding recipes + mapping of items per recipe

CREATE TABLE recipe(
    id                  INT AUTO_INCREMENT  NOT NULL,
    name                VARCHAR(256)        NOT NULL,
    descriptionFile     VARCHAR(256)        NOT NULL,
    createdBy           BIGINT              NOT NULL,
    defaultQuantity     INT                 NOT NULL,
    PRIMARY KEY (`id`)
);

CREATE TABLE itemsPerRecipe(
    id          INT AUTO_INCREMENT  NOT NULL,
    recipeId    INT                 NOT NULL,
    itemId      INT                 NOT NULL,
    quantity    INT                 NOT NULL,
    PRIMARY KEY (`id`)
);

CREATE TABLE sharedRecipe(
    id          INT AUTO_INCREMENT  NOT NULL,
    recipeId    INT                 NOT NULL,
    sharedWith  BIGINT              NOT NULL,
    PRIMARY KEY (`id`)
);

-- Keeping track of the shopping history to suggest items

CREATE TABLE history(
    id              BIGINT AUTO_INCREMENT   NOT NULL,
    itemId          INT                     NOT NULL,
    totalQuantity   INT                     NOT NULL,
    since           DATETIME                NOT NULL,
    weeklyUse       INT                     NOT NULL,
    PRIMARY KEY (`id`)
);