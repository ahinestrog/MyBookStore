PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS carts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL UNIQUE,         -- 1 carrito activo por usuario
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS cart_items (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  cart_id INTEGER NOT NULL,
  book_id INTEGER NOT NULL,
  title TEXT NOT NULL,
  unit_price_cents INTEGER NOT NULL,       -- precio en centavos
  qty INTEGER NOT NULL CHECK (qty > 0),
  UNIQUE(cart_id, book_id),
  FOREIGN KEY(cart_id) REFERENCES carts(id) ON DELETE CASCADE
);

CREATE TRIGGER IF NOT EXISTS carts_updated_at
AFTER UPDATE ON carts
BEGIN
  UPDATE carts SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
