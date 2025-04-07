-- +goose Up
CREATE TABLE video (
  id INTERGER PRIMARY KEY NOT NULL,
  name text NOT NULL,
  uploaded text NOT NULL
);

-- +goose Down
DROP TABLE video;
