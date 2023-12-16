DROP TABLE IF EXISTS shoppers;
DROP TABLE IF EXISTS items;
DROP TABLE IF EXISTS shoppinglists;

CREATE TABLE shoppers (
    id          SERIAL          PRIMARY KEY,
    name        VARCHAR(128)    NOT NULL,
    favRecipe   INT
);

CREATE TABLE items (
    id      SERIAL          PRIMARY KEY,
    name    VARCHAR(256)    NOT NULL,
    image   VARCHAR(256)    NOT NULL
);

CREATE TABLE shoppinglists (
    id          SERIAL  PRIMARY KEY,
    listId      INT     NOT NULL,
    itemId      INT     NOT NULL,
    quantity    INT     NOT NULL
);

-- TODO: Creating the list of lists? Should be clear from the list table already, shouldn't it?

-- Apply this file the following way:
-- Log into the database with postgres: psql -U <user>
-- Execute this file: \i ./create_database.sql

-- To switch the database:
-- \c <database name>

-- To show the tables:
-- \dt

-- To show the content of a table:
-- SELECT * FROM <table>;

-- To insert data into a table:
-- INSERT INTO <table> (<col1>, <col2>, <col3>, ...) VALUES (<val1>, <val2>, <val3>, ...);
-- IMPORTANT: VARCHAR must be signaled using ' instead of "
-- IMPORTANT: The (<col1>, ...) can also be omitted if value for all columns are given