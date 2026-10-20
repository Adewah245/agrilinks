-- =============================================================
-- 039_learner_role.sql
--
-- Adds the learner role defined by the Agro-Shield user model.
-- Learning remains available to every authenticated user, while
-- users whose primary intent is learning receive the learner role.
-- =============================================================

ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_role_check;

ALTER TABLE users
    ADD CONSTRAINT users_role_check
    CHECK (
        role IN (
            'farmer',
            'buyer',
            'learner',
            'ministry',
            'organisation'
        )
    );
