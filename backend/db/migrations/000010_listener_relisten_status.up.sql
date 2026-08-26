-- Retcon listener impact: add relisten_status to listening_progress.
ALTER TABLE listening_progress
    ADD COLUMN relisten_status TEXT NOT NULL DEFAULT 'NO_RELISTEN_NEEDED';
