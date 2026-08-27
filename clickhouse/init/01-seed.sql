CREATE DATABASE IF NOT EXISTS demo;

CREATE TABLE IF NOT EXISTS demo.events (
    ts     DateTime,
    user   String,
    action String
) ENGINE = MergeTree ORDER BY ts;

INSERT INTO demo.events
SELECT now() - number,
       ['ana', 'bob', 'cid'][number % 3 + 1],
       ['login', 'query', 'logout'][number % 3 + 1]
FROM numbers(1000);

-- SQL-defined roles and users: these are the ones the GUI can actually manage.
CREATE ROLE IF NOT EXISTS reader;
GRANT SELECT ON demo.* TO reader;

CREATE ROLE IF NOT EXISTS writer;
GRANT SELECT, INSERT ON demo.* TO writer;

CREATE USER IF NOT EXISTS analyst IDENTIFIED WITH plaintext_password BY 'analyst';
GRANT reader TO analyst;

CREATE USER IF NOT EXISTS etl IDENTIFIED WITH plaintext_password BY 'etl';
GRANT writer TO etl;

-- Something for the Policies screen to display.
CREATE ROW POLICY IF NOT EXISTS ana_only ON demo.events FOR SELECT USING user = 'ana' TO analyst;
CREATE QUOTA IF NOT EXISTS light FOR INTERVAL 1 hour MAX queries = 100 TO analyst;
