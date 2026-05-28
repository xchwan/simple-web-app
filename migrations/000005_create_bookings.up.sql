CREATE TABLE IF NOT EXISTS bookings (
    id        INT                              NOT NULL AUTO_INCREMENT,
    user_id   INT                              NOT NULL,
    ticket_id INT                              NOT NULL,
    wallet_id INT                              NOT NULL,
    status    ENUM('confirmed', 'cancelled')   NOT NULL DEFAULT 'confirmed',
    booked_at DATETIME(3)                      NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE INDEX idx_bookings_ticket_id (ticket_id),
    INDEX idx_bookings_user_id (user_id),
    INDEX idx_bookings_wallet_id (wallet_id)
);
