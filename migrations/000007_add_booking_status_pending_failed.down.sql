ALTER TABLE bookings MODIFY COLUMN status ENUM('confirmed','cancelled') NOT NULL DEFAULT 'confirmed';
