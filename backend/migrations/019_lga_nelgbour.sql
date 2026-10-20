-- ============================================
-- AGRO-SHIELD
-- Migration 018: LGA Neighbors
-- ============================================

CREATE TABLE IF NOT EXISTS lga_neighbors (
    lga_id INTEGER NOT NULL,
    neighbor_lga_id INTEGER NOT NULL,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_lga_neighbors_lga
        FOREIGN KEY (lga_id)
        REFERENCES lgas(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_lga_neighbors_neighbor
        FOREIGN KEY (neighbor_lga_id)
        REFERENCES lgas(id)
        ON DELETE CASCADE,

    CONSTRAINT unique_lga_neighbor
        UNIQUE (lga_id, neighbor_lga_id),

    CONSTRAINT chk_lga_neighbors_not_same
        CHECK (lga_id <> neighbor_lga_id)
);

CREATE INDEX IF NOT EXISTS idx_lga_neighbors_lga_id
    ON lga_neighbors(lga_id);

CREATE INDEX IF NOT EXISTS idx_lga_neighbors_neighbor_lga_id
    ON lga_neighbors(neighbor_lga_id);