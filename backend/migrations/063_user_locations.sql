-- Keep existing location text for old records and display. New and updated
-- profiles link to the same state/LGA/community records used by products.
ALTER TABLE users ADD COLUMN state_id INTEGER REFERENCES states(id);
ALTER TABLE users ADD COLUMN lga_id INTEGER REFERENCES lgas(id);
ALTER TABLE users ADD COLUMN community_id INTEGER REFERENCES communities(id);
ALTER TABLE lgas ADD CONSTRAINT lgas_id_state_unique UNIQUE(id,state_id);
ALTER TABLE communities ADD CONSTRAINT communities_id_lga_unique UNIQUE(id,lga_id);
ALTER TABLE users ADD CONSTRAINT users_lga_state FOREIGN KEY(lga_id,state_id) REFERENCES lgas(id,state_id);
ALTER TABLE users ADD CONSTRAINT users_community_lga FOREIGN KEY(community_id,lga_id) REFERENCES communities(id,lga_id);
ALTER TABLE users ADD CONSTRAINT users_location_parent_check CHECK(
    (lga_id IS NULL OR state_id IS NOT NULL) AND (community_id IS NULL OR lga_id IS NOT NULL));

UPDATE users u SET state_id=s.id FROM states s WHERE LOWER(BTRIM(u.state))=LOWER(s.name);
UPDATE users u SET lga_id=l.id FROM lgas l WHERE l.state_id=u.state_id AND LOWER(BTRIM(u.lga))=LOWER(l.name);
INSERT INTO communities(lga_id,name) SELECT DISTINCT lga_id,BTRIM(location) FROM users
    WHERE lga_id IS NOT NULL AND BTRIM(location)<>'' ON CONFLICT(lga_id,name) DO NOTHING;
UPDATE users u SET community_id=c.id FROM communities c WHERE c.lga_id=u.lga_id AND c.name=BTRIM(u.location);
