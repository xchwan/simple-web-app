ALTER TABLE bookings MODIFY COLUMN status ENUM('pending','confirmed','cancelled','failed') NOT NULL DEFAULT 'confirmed';
