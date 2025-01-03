-- Execute this file with the following command:
-- Log into mysql: 
-- sudo mysql
-- source ./create_mysql_db.sql;
-- OR directly:
-- sudo mysql < ./create_mysql_db.sql

CREATE DATABASE IF NOT EXISTS shoppinglist;

use shoppinglist;

DROP USER IF EXISTS '<username>'@'<locality>';
CREATE USER IF NOT EXISTS '<username>'@'<locality>' IDENTIFIED BY '<password>';

SELECT User
FROM mysql.user;

GRANT ALL PRIVILEGES ON shoppinglist.* TO '<username>'@'<locality>' WITH GRANT OPTION;

FLUSH PRIVILEGES;

SHOW GRANTS FOR '<username>'@'<locality>';

-- Start creating the database tables

DROP TABLE IF EXISTS shoppers; -- The users of our system
DROP TABLE IF EXISTS items; -- The items that can be shopped and shared
DROP TABLE IF EXISTS shoppinglist; -- Holding generic list information
DROP TABLE IF EXISTS sharedList; -- Which user can access which list
DROP TABLE IF EXISTS itemsPerList; -- The mapping of items to lists
DROP TABLE IF EXISTS recipe; -- The description of recipes
DROP TABLE IF EXISTS ingredientPerRecipe; -- The items that make a recipe
DROP TABLE IF EXISTS descriptionPerRecipe; -- The descriptions that make a recipe
DROP TABLE IF EXISTS sharedRecipe; -- Which user can access which recipe
DROP TABLE IF EXISTS history;
-- What has been bought in the past

-- Table for AUTHENTICATION + AUTHORIZATION (mapping what lists / items can be seen)

CREATE TABLE shoppers
(
    id        BIGINT       NOT NULL,
    username  VARCHAR(256) NOT NULL,
    passwd    VARCHAR(512) NOT NULL,
    created   DATETIME     NOT NULL,
    lastLogin DATETIME     NOT NULL,
    PRIMARY KEY (`id`)
);

-- Table for Items that can be shopped. Shared among all users because currently only shared via name

CREATE TABLE items
(
    id   INT AUTO_INCREMENT NOT NULL,
    name VARCHAR(256)       NOT NULL,
    icon VARCHAR(256)       NOT NULL,
    PRIMARY KEY (`id`)
);

-- Table holding the list information + the mapping of items to lists

CREATE TABLE shoppinglist
(
    id         INT AUTO_INCREMENT NOT NULL,
    listId     BIGINT             NOT NULL,
    name       VARCHAR(256)       NOT NULL,
    createdBy  BIGINT             NOT NULL,
    created    DATETIME           NOT NULL,
    lastEdited DATETIME           NOT NULL,
    version    BIGINT             NOT NULL,
    PRIMARY KEY (`id`)
);

CREATE TABLE sharedList
(
    id           INT AUTO_INCREMENT NOT NULL,
    listId       BIGINT             NOT NULL,
    createdBy    BIGINT             NOT NULL,
    sharedWithId BIGINT             NOT NULL,
    created      DATETIME           NOT NULL,
    PRIMARY KEY (`id`)
);

CREATE TABLE itemsPerList
(
    id        INT AUTO_INCREMENT NOT NULL,
    listId    BIGINT             NOT NULL,
    itemId    INT                NOT NULL,
    quantity  INT                NOT NULL,
    checked   BOOLEAN            NOT NULL,
    createdBy BIGINT             NOT NULL,
    addedBy   BIGINT             NOT NULL,
    PRIMARY KEY (`id`)
);

-- Table holding recipes + mapping of items per recipe

CREATE TABLE recipe
(
    recipeId       BIGINT       NOT NULL,
    createdBy      BIGINT       NOT NULL,
    name           VARCHAR(256) NOT NULL,
    createdAt      DATETIME     NOT NULL,
    lastUpdate     DATETIME     NOT NULL,
    version        INT          NOT NULL,
    defaultPortion INT          NOT NULL,
    PRIMARY KEY (recipeId, createdBy)
);

CREATE TABLE ingredientPerRecipe
(
    recipeId     INT         NOT NULL,
    createdBy    BIGINT      NOT NULL,
    itemId       INT         NOT NULL,
    quantity     INT         NOT NULL,
    quantityType VARCHAR(32) NOT NULL,
    PRIMARY KEY (recipeId, createdBy, itemId)
);

CREATE TABLE descriptionPerRecipe
(
    recipeId         BIGINT         NOT NULL,
    createdBy        BIGINT         NOT NULL,
    description      VARCHAR(16000) NOT NULL,
    descriptionOrder INT            NOT NULL,
    PRIMARY KEY (recipeId, createdBy, descriptionOrder)
);

CREATE TABLE sharedRecipe
(
    recipeId   BIGINT NOT NULL,
    createdBy  BIGINT NOT NULL,
    sharedWith BIGINT NOT NULL,
    PRIMARY KEY (recipeId, createdBy, sharedWith)
);

-- Keeping track of the shopping history to suggest items

CREATE TABLE history
(
    id            BIGINT AUTO_INCREMENT NOT NULL,
    itemId        INT                   NOT NULL,
    totalQuantity INT                   NOT NULL,
    since         DATETIME              NOT NULL,
    weeklyUse     INT                   NOT NULL,
    PRIMARY KEY (`id`)
);