DROP TRIGGER IF EXISTS audit_events_append_only ON audit_events;
DROP FUNCTION IF EXISTS synaudio_reject_audit_mutation();
DROP TABLE IF EXISTS audit_events;
