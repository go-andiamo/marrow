CREATE TABLE people (
    id          VARCHAR(36),
    given_name  VARCHAR(32),
    family_name VARCHAR(32),
    dob         TIMESTAMP(3)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;