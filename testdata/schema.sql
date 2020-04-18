CREATE TABLE Singers (
SingerId   INT64 NOT NULL,
FirstName  STRING(1024),
LastName   STRING(1024),
SingerInfo BYTES(MAX)
) PRIMARY KEY (SingerId);

CREATE TABLE Albums (
SingerId     INT64 NOT NULL,
AlbumId      INT64 NOT NULL,
AlbumTitle   STRING(MAX)
) PRIMARY KEY (SingerId, AlbumId),
INTERLEAVE IN PARENT Singers ON DELETE CASCADE;

CREATE TABLE Examples (
  ID STRING(1024),
  LastUpdateTime  TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (ID)
