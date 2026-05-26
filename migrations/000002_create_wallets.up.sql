CREATE TABLE wallets (
    id      INT            NOT NULL AUTO_INCREMENT,
    user_id INT            NOT NULL,
    name    VARCHAR(255)   NOT NULL,
    balance DECIMAL(15, 2) NOT NULL DEFAULT 0,
    PRIMARY KEY (id),
    INDEX idx_wallets_user_id (user_id)
);
