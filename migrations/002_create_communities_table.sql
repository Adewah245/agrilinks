CREATE TABLE communities (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    lga_id INT NOT NULL,
    name VARCHAR(100) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_communities_lga
        FOREIGN KEY (lga_id)
        REFERENCES lgas(id)
        ON DELETE RESTRICT,

    CONSTRAINT unique_community_per_lga
        UNIQUE (lga_id, name)
);