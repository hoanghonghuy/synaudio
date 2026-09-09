CREATE TABLE auth_abuse_counters (
    scope TEXT NOT NULL,
    key_hash TEXT NOT NULL,
    window_start TIMESTAMPTZ NOT NULL,
    count INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (scope, key_hash, window_start)
);

CREATE INDEX auth_abuse_counters_window_start_idx
    ON auth_abuse_counters (window_start);
