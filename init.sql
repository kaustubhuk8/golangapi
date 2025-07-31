USE manifold;


CREATE TABLE IF NOT EXISTS users (
    user_id VARCHAR(255) PRIMARY KEY,
    words_left INT NOT NULL DEFAULT 1000000,
    total_words INT NOT NULL DEFAULT 1000000,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_words_left (words_left)
) ENGINE=InnoDB;


CREATE TABLE IF NOT EXISTS requests (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    data TEXT,
    duration INT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_user_id (user_id),
    INDEX idx_created_at (created_at),
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE
) ENGINE=InnoDB;

INSERT IGNORE INTO users (user_id, words_left, total_words) VALUES 
('user1', 1000000, 1000000),
('user2', 1000000, 1000000),
('user3', 1000000, 1000000),
('user4', 1000000, 1000000),
('user5', 1000000, 1000000),
('user6', 1000000, 1000000),
('user7', 1000000, 1000000),
('user8', 1000000, 1000000),
('user9', 1000000, 1000000),
('user10', 1000000, 1000000);

SET GLOBAL innodb_buffer_pool_size = 1073741824; -- 1GB
SET GLOBAL max_connections = 1000;
SET GLOBAL thread_cache_size = 200;