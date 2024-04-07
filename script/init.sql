CREATE TABLE IF NOT EXISTS banner
(
    id         bigserial   not null primary key,
    content    jsonb       not null,
    is_active  boolean     not null default true,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

CREATE TABLE IF NOT EXISTS features_tags_banner
(
    id         bigserial not null primary key,
    banner_id  bigint    not null references banner (id) on delete cascade,
    tag_id     bigint    not null,
    feature_id bigint    not null,
    --   version    bigint    not null,
    constraint banner_identifier UNIQUE (tag_id, feature_id)
);

-- Для обновления поля update_at после обновление таблицы banner
CREATE OR REPLACE FUNCTION banner_update_trigger() RETURNS TRIGGER AS
$$
BEGIN
    UPDATE banner SET updated_at = now() WHERE id = NEW.id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER banner_update
    AFTER UPDATE
    ON banner
    FOR EACH ROW
    WHEN (OLD.* IS DISTINCT FROM NEW.*)
EXECUTE FUNCTION banner_update_trigger();

