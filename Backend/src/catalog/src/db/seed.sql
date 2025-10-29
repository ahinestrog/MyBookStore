INSERT INTO books (title, author, price_cents, cover_url, created_unix) VALUES
('Leonardo Da Vinci: La Biografía','Walter Isaacson', 50000, '/images/img-ld-labiografia.jpeg', strftime('%s','now')-86400),
('Inteligencia Genial','Michael Gelb', 30000, '/images/img-ld-inteligenciagenial.jpeg', strftime('%s','now')-3600),
('Clean Code','Robert C. Martin', 120000, '', strftime('%s','now')-7200),
('The Pragmatic Programmer','Andy Hunt / Dave Thomas', 115000, '', strftime('%s','now')-82000),
('Design Patterns','Erich Gamma et al.', 135000, '', strftime('%s','now')-172800),
('Refactoring','Martin Fowler', 125000, '', strftime('%s','now')-259200),
('Introduction to Algorithms','Cormen, Leiserson, Rivest, Stein', 220000, '', strftime('%s','now')-300000),
('You Don''t Know JS','Kyle Simpson', 90000, '', strftime('%s','now')-400000),
('Effective Java','Joshua Bloch', 140000, '', strftime('%s','now')-500000),
('Operating Systems: Design and Implementation','Andrew S. Tanenbaum', 180000, '', strftime('%s','now')-600000),
('Distributed Systems','Andrew S. Tanenbaum', 160000, '', strftime('%s','now')-700000),
('Deep Learning','Ian Goodfellow, Yoshua Bengio, Aaron Courville', 250000, '', strftime('%s','now')-800000);
