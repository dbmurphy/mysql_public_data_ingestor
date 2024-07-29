CREATE DATABASE IF NOT EXISTS testdb;
USE testdb;

CREATE TABLE IF NOT EXISTS flights (
    time INT,
    icao24 VARCHAR(10),
    callsign VARCHAR(10),
    origin_country VARCHAR(50),
    time_position INT,
    last_contact INT,
    longitude FLOAT,
    latitude FLOAT,
    baro_altitude FLOAT,
    on_ground BOOLEAN,
    velocity FLOAT,
    true_track FLOAT,
    vertical_rate FLOAT,
    sensors JSON,
    geo_altitude FLOAT,
    squawk VARCHAR(10),
    spi BOOLEAN,
    position_source INT
);
