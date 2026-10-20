-- =============================================================
-- 053_chat_read_state.sql
--
-- Tracks how far each conversation member has read. A message is
-- considered seen when another member's last_read_at is at or
-- after the message's created_at. The same timestamp powers unread
-- counts in the conversation list.
-- =============================================================

ALTER TABLE conversation_members
    ADD COLUMN IF NOT EXISTS last_read_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conrelid = 'conversation_members'::regclass
          AND conname = 'conversation_members_last_read_check'
    ) THEN
        ALTER TABLE conversation_members
            ADD CONSTRAINT conversation_members_last_read_check
            CHECK (last_read_at >= joined_at);
    END IF;
END
$$;
