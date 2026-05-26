CREATE TABLE events (
    id           INT          NOT NULL AUTO_INCREMENT,
    organizer_id INT          NOT NULL,
    name         VARCHAR(255) NOT NULL,
    description  TEXT,
    start_at     DATETIME(3)  NOT NULL,
    PRIMARY KEY (id),
    INDEX idx_events_organizer_id (organizer_id)
);
