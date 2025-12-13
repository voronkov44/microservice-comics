CREATE TABLE IF NOT EXISTS favorites (
    user_id BIGINT NOT NULL,
    comic_id BIGINT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, comic_id)
);