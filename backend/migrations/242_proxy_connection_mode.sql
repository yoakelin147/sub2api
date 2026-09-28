ALTER TABLE proxies ADD COLUMN IF NOT EXISTS connection_mode varchar(20) NOT NULL DEFAULT 'reuse';

ALTER TABLE proxies ADD CONSTRAINT proxies_connection_mode_valid
    CHECK (connection_mode = 'reuse' OR (connection_mode = 'per_request' AND protocol IN ('http', 'https')));
