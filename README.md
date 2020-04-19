# Ratchet

ratchet is a simple CLI tool for accessing Cloud Spanner. This tool allows you to
throw queries to the Cloud Spanner (or an emulator of that).

If `SPANNER_EMULATOR_HOST` env is set, ratchet uses it.

```
Usage:
  ratchet [command]

Available Commands:
  describe    Show Database DDLs.
  exec        Throw specified SQL(s) to Cloud Spanner.
  help        Help about any command

Flags:
  -d, --database SPANNER_DATABASE_ID   your-database-id. (you can also set by SPANNER_DATABASE_ID)
  -h, --help                           help for ratchet
  -i, --instance SPANNER_INSTANCE_ID   your-instance-id. (you can also set by SPANNER_INSTANCE_ID)
  -p, --project SPANNER_PROJECT_ID     your-project-id. (you can also set by SPANNER_PROJECT_ID)
```

# Installation

```
go install github.com/matope/ratchet
```

# How to use

```bash
# you can also set by -p, -i, -d
$ export SPANNER_PROJECT_ID=<your-project-id>
$ export SPANNER_INSTANCE_ID=<your-instance-id>
$ export SPANNER_DATABASE_ID=<your-database-id>

# Set if you use spanner-emulator such like handy-spanner.
$ export SPANNER_EMULATOR_HOST=localhost:9999

$ ratchet exec "SELECT * FROM information_schema.tables"
db: projects/fake/instances/fake/databases/fake
SPANNER_EMULATOR_HOST: localhost:9999

sql:SELECT * FROM information_schema.tables
+---------------+--------------------+----------------+-------------------+------------------+---------------+
| TABLE_CATALOG |    TABLE_SCHEMA    |   TABLE_NAME   | PARENT_TABLE_NAME | ON_DELETE_ACTION | SPANNER_STATE |
+---------------+--------------------+----------------+-------------------+------------------+---------------+
|               | INFORMATION_SCHEMA | SCHEMATA       |                   |                  |               |
|               | INFORMATION_SCHEMA | TABLES         |                   |                  |               |
|               | INFORMATION_SCHEMA | COLUMNS        |                   |                  |               |
|               | INFORMATION_SCHEMA | INDEXES        |                   |                  |               |
|               | INFORMATION_SCHEMA | INDEX_COLUMNS  |                   |                  |               |
|               | INFORMATION_SCHEMA | COLUMN_OPTIONS |                   |                  |               |
|               |                    | Singers        |                   |                  | COMMITTED     |
|               |                    | Albums         | Singers           | CASCADE          | COMMITTED     |
|               |                    | Examples       |                   |                  | COMMITTED     |
+---------------+--------------------+----------------+-------------------+------------------+---------------+
9 record(s) found.
```

## Describe Database

Using `describe` command, you can get Database DDL(s). (For now, handy-spanner does not yet implement it)

```
$ ratchet -p <PROJECT_ID> -i <INSTANCE_ID> -d <DATABASE> describe
Found 3 DDL(s)

CREATE TABLE Examples (
  ID STRING(1024),
  LastUpdateTime TIMESTAMP NOT NULL OPTIONS (
    allow_commit_timestamp = true
  ),
) PRIMARY KEY(ID)

CREATE TABLE Singers (
  SingerId INT64 NOT NULL,
  FirstName STRING(1024),
  LastName STRING(1024),
  SingerInfo BYTES(MAX),
) PRIMARY KEY(SingerId)

CREATE TABLE Albums (
  SingerId INT64 NOT NULL,
  AlbumId INT64 NOT NULL,
  AlbumTitle STRING(MAX),
) PRIMARY KEY(SingerId, AlbumId),
  INTERLEAVE IN PARENT Singers ON DELETE CASCADE
```

## Execute SQL(s).

Using `exec` command, you can throw queries and DDL/DML SQL to Cloud Spanner.

```
$ ratchet -p <PROJECT_ID> -i <INSTANCE_ID> -d <DATABASE> exec "SELECT * From Singers; SELECT * FROM Albums"

sql:SELECT * From Singers
+----------+-----------+----------+------------+
| SingerId | FirstName | LastName | SingerInfo |
+----------+-----------+----------+------------+
|        1 | Marc      | Richards |            |
|        2 | Catalina  | Smith    |            |
|        3 | Alice     | Trentor  |            |
|        4 | Lea       | Martin   |            |
|        5 | David     | Lomond   |            |
+----------+-----------+----------+------------+
5 record(s) found.

sql:SELECT * FROM Albums
+----------+---------+-------------------------+
| SingerId | AlbumId |       AlbumTitle        |
+----------+---------+-------------------------+
|        1 |       1 | Total Junk              |
|        1 |       2 | Go, Go, Go              |
|        2 |       1 | Green                   |
|        2 |       2 | Forever Hold Your Peace |
|        2 |       3 | Terrified               |
+----------+---------+-------------------------+
5 record(s) found.
```

## Execute SQL(s) from a file.

```
$ cat testdata/inserts.sql
INSERT INTO Singers(SingerId, FirstName, LastName) VALUES (1, "Marc", "Richards");
INSERT INTO Singers(SingerId, FirstName, LastName) VALUES (2, "Catalina", "Smith");
INSERT INTO Albums(SingerId, AlbumId, AlbumTitle) VALUES(1, 1, "Total Junk");
INSERT INTO Albums(SingerId, AlbumId, AlbumTitle) VALUES(1, 2, "Go, Go, Go");

$ ratchet -p <PROJECT_ID> -i <INSTANCE_ID> -d <DATABASE> exec --file ./testdata/inserts.sql
```
