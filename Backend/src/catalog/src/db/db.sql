PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS books (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  title         TEXT NOT NULL,
  author        TEXT NOT NULL,
  price_cents   INTEGER NOT NULL DEFAULT 0,  -- common.Money.cents
  cover_url     TEXT DEFAULT '',
  created_unix  INTEGER NOT NULL             -- segundos Unix
);

CREATE INDEX IF NOT EXISTS idx_books_title  ON books(title);
CREATE INDEX IF NOT EXISTS idx_books_author ON books(author);
