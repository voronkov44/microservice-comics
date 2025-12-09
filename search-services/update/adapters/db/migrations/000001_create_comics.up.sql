-- Таблица comics - id, картинка и нормализованные слова
CREATE TABLE IF NOT EXISTS comics (
    id          INT PRIMARY KEY,
    img_url     TEXT NOT NULL,
    title       TEXT[] NOT NULL,
    alt         TEXT[] NOT NULL,
    words       TEXT[] NOT NULL,
    fetched_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);