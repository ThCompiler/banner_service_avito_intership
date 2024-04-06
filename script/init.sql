CREATE TABLE IF NOT EXISTS features_tags_banner
(
    id         bigserial not null primary key,
    banner_id  bigint    not null references banner (id) on delete cascade,
    tag_id     bigint    not null,
    feature_id bigint    not null,
    --   version    bigint    not null,
    constraint banner_identifier UNIQUE (tag_id, feature_id)
);

CREATE TABLE IF NOT EXISTS banner
(
    id         bigserial   not null primary key,
    content    jsonb       not null,
    is_active  boolean     not null default true,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

WITH selected_banner as (
    SELECT banner_id FROM features_tags_banner
    WHERE (CASE WHEN $1 IS NOT NULL THEN feature_id = $1 ELSE true END)
      and (CASE WHEN $2 IS NOT NULL THEN tag_id = $2 ELSE true END)
)
SELECT content FROM banner inner join selected_banner on (selected_banner.banner_id = banner.id)
LIMIT $3 OFFSET $4

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
