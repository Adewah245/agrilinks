-- Learning communities reuse normal conversations and membership checks.
-- Only groups explicitly registered here are open for learners to join.
CREATE TABLE learning_rooms (
    conversation_id INTEGER PRIMARY KEY REFERENCES conversations(id) ON DELETE CASCADE,
    farming_type_id INTEGER NOT NULL REFERENCES farming_types(id),
    description TEXT CHECK (description IS NULL OR BTRIM(description) <> ''),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX learning_rooms_farming_type_idx ON learning_rooms(farming_type_id);
