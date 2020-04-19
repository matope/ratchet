# Ratchet

A simple client CLI for Cloud Spanner

ratchet is a simple CLI tool for accessing Cloud Spanner. This tool allows you to
throw queries to the Cloud Spanner (or an emulator of that).

If `SPANNER_EMULATOR_HOST` env is set, ratchet uses it.

``
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

# Example

## Describe Database

Using describe command, you can get Database DDL(s). (For now, handy-spanner does not yet implement it)

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

You can throw queries and DDL/DML SQL to Cloud Spanner.

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
ratchet -p <PROJECT_ID> -i <INSTANCE_ID> -d <DATABASE> exec --file ./testdata/schema.sql 
```
