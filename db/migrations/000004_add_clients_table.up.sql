CREATE TABLE IF NOT EXISTS clients (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    client_id VARCHAR(255) NOT NULL,
    client_secret_hash VARCHAR(255) NOT NULL,
    client_type ENUM('public', 'confidential') NOT NULL,
    app_name VARCHAR(32),
    UNIQUE(client_id),
);

