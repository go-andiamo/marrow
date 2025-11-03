-- Function uuidV4() generates a Version 4 UUID
-- The default MySql uuid() function generates a Version 1 (which isn't very secure as it leaks MAC address)
-- original code from https://stackoverflow.com/questions/32965743/how-to-generate-a-uuidv4-in-mysql
-- NOTE: this function CANNOT be used for column default values!!!  But can be used for seeding
CREATE FUNCTION uuidV4()
    RETURNS CHAR(36)  NO SQL
BEGIN
    -- Build the complete UUID Version 4
RETURN LOWER(CONCAT(
        HEX(RANDOM_BYTES(4)), '-',
        HEX(RANDOM_BYTES(2)), '-4',
        SUBSTR(HEX(RANDOM_BYTES(2)), 2, 3), '-',
        HEX(FLOOR(ASCII(RANDOM_BYTES(1)) / 64)+8), SUBSTR(HEX(RANDOM_BYTES(2)), 2, 3), '-',
        HEX(RANDOM_BYTES(6))));
END;
