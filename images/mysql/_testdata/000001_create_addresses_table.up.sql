CREATE TABLE addresses (
    id        VARCHAR(36),
    person_id VARCHAR(36),
    address   TEXT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;