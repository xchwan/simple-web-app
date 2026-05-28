CREATE TABLE IF NOT EXISTS tickets (
    id       INT                          NOT NULL AUTO_INCREMENT,
    event_id INT                          NOT NULL,
    seat     VARCHAR(50)                  NOT NULL,
    price    DECIMAL(10, 2)               NOT NULL,
    status   ENUM('available', 'sold')    NOT NULL DEFAULT 'available',
    PRIMARY KEY (id),
    INDEX idx_tickets_event_id (event_id)
);
