-- Execute this file with the following command:
-- Log into mysql: 
-- sudo mysql
-- source ./create_mysql_db.sql;
-- OR directly:
-- sudo mysql < ./create_mysql_db.sql

DROP DATABASE IF EXISTS shopping_list_prod;
CREATE DATABASE shopping_list_prod;

use shopping_list_prod;

DROP USER IF EXISTS '<username>'@'%';
CREATE USER IF NOT EXISTS '<username>'@'%' IDENTIFIED BY '<password>';

SELECT User
FROM mysql.user;

GRANT ALL PRIVILEGES ON shopping_list_prod.* TO '<username>'@'%' WITH GRANT OPTION;
GRANT FILE ON *.* TO '<username>'@'%' WITH GRANT OPTION;

FLUSH PRIVILEGES;

SHOW GRANTS FOR '<username>'@'%';

-- Table for AUTHENTICATION + AUTHORIZATION (mapping what lists / items can be seen)

CREATE TABLE shoppers
(
    id        BIGINT       NOT NULL,
    username  VARCHAR(128) NOT NULL,
    passwd    VARCHAR(512) NOT NULL,
    created   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    lastLogin DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
);

CREATE TABLE role
(
    user_id BIGINT     NOT NULL,
    role    VARCHAR(2) NOT NULL DEFAULT 'US',
    PRIMARY KEY (user_id, role),
    FOREIGN KEY (user_id) REFERENCES shoppers (id) ON DELETE CASCADE
);

-- Table for Items that can be shopped. Shared among all users because currently only shared via name

CREATE TABLE items
(
    id   BIGINT AUTO_INCREMENT NOT NULL,
    name VARCHAR(256)          NOT NULL,
    icon VARCHAR(256)          NOT NULL,
    PRIMARY KEY (id)
);

-- Table holding the list information + the mapping of items to lists

CREATE TABLE shopping_list
(
    listId     BIGINT       NOT NULL,
    createdBy  BIGINT       NOT NULL,
    name       VARCHAR(256) NOT NULL,
    created    DATETIME     NOT NULL,
    lastEdited DATETIME     NOT NULL,
    version    BIGINT       NOT NULL DEFAULT 1,
    PRIMARY KEY (listId, createdBy),
    FOREIGN KEY (createdBy) REFERENCES shoppers (id) ON DELETE CASCADE
);

CREATE TABLE items_per_list
(
    listId    BIGINT  NOT NULL,
    createdBy BIGINT  NOT NULL,
    itemId    BIGINT  NOT NULL,
    quantity  INT     NOT NULL,
    checked   BOOLEAN NOT NULL,
    addedBy   BIGINT,
    PRIMARY KEY (listId, createdBy, itemId),
    FOREIGN KEY (listId, createdBy) REFERENCES shopping_list (listId, createdBy) ON DELETE CASCADE,
    FOREIGN KEY (itemId) REFERENCES items (id) ON DELETE CASCADE,
    FOREIGN KEY (addedBy) REFERENCES shoppers (id) ON DELETE SET NULL
);

CREATE TABLE shared_list
(
    listId       BIGINT   NOT NULL,
    createdBy    BIGINT   NOT NULL,
    sharedWithId BIGINT   NOT NULL,
    created      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (listId, createdBy, sharedWithId),
    FOREIGN KEY (listId, createdBy) REFERENCES shopping_list (listId, createdBy) ON DELETE CASCADE,
    FOREIGN KEY (sharedWithId) REFERENCES shoppers (id) ON DELETE CASCADE
);

-- Table holding recipes + mapping of items per recipe

CREATE TABLE recipe
(
    recipeId       INT          NOT NULL,
    createdBy      BIGINT       NOT NULL,
    name           VARCHAR(256) NOT NULL,
    createdAt      DATETIME     NOT NULL,
    lastUpdate     DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    version        INT          NOT NULL DEFAULT 1,
    defaultPortion INT          NOT NULL,
    PRIMARY KEY (recipeId, createdBy),
    FOREIGN KEY (createdBy) REFERENCES shoppers (id) ON DELETE CASCADE
);

CREATE TABLE ingredient_per_recipe
(
    recipeId     INT         NOT NULL,
    createdBy    BIGINT      NOT NULL,
    itemId       BIGINT      NOT NULL,
    quantity     INT         NOT NULL,
    quantityType VARCHAR(32) NOT NULL DEFAULT 'PCS',
    PRIMARY KEY (recipeId, createdBy, itemId),
    FOREIGN KEY (recipeId, createdBy) REFERENCES recipe (recipeId, createdBy) ON DELETE CASCADE,
    FOREIGN KEY (itemId) REFERENCES items (id) ON DELETE CASCADE
);

CREATE TABLE description_per_recipe
(
    recipeId         INT           NOT NULL,
    createdBy        BIGINT        NOT NULL,
    description      VARCHAR(1000) NOT NULL,
    descriptionOrder INT           NOT NULL,
    PRIMARY KEY (recipeId, createdBy, descriptionOrder),
    FOREIGN KEY (recipeId, createdBy) REFERENCES recipe (recipeId, createdBy) ON DELETE CASCADE
);

CREATE TABLE images_per_recipe
(
    recipeId  INT         NOT NULL,
    createdBy BIGINT      NOT NULL,
    filename  VARCHAR(50) NOT NULL,
    PRIMARY KEY (recipeId, createdBy, filename),
    FOREIGN KEY (recipeId, createdBy) REFERENCES recipe (recipeId, createdBy) ON DELETE CASCADE
);

CREATE TABLE shared_recipe
(
    recipeId   INT    NOT NULL,
    createdBy  BIGINT NOT NULL,
    sharedWith BIGINT NOT NULL,
    PRIMARY KEY (recipeId, createdBy, sharedWith),
    FOREIGN KEY (recipeId, createdBy) REFERENCES recipe (recipeId, createdBy) ON DELETE CASCADE,
    FOREIGN KEY (sharedWith) REFERENCES shoppers (id) ON DELETE CASCADE
);

-- Keeping track of the shopping history to suggest items

CREATE TABLE history
(
    id            BIGINT AUTO_INCREMENT NOT NULL,
    itemId        BIGINT                NOT NULL,
    totalQuantity INT                   NOT NULL,
    since         DATETIME              NOT NULL,
    weeklyUse     INT                   NOT NULL,
    PRIMARY KEY (`id`),
    FOREIGN KEY (itemId) REFERENCES items (id) ON DELETE CASCADE
);