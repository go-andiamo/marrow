CREATE TABLE pets (
    id          VARCHAR(36) NOT NULL PRIMARY KEY,
    name        VARCHAR(32) NOT NULL,
    dob         DATE        NOT NULL,
    category_id VARCHAR(36) NOT NULL,
    CONSTRAINT fk_category_id FOREIGN KEY (category_id) REFERENCES categories(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
