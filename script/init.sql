CREATE TABLE IF NOT EXISTS banner
(
    id           bigserial   not null primary key,
    is_active    boolean     not null default true,
    created_at   timestamptz not null default now(), -- время появление банера как такового
    updated_at   timestamptz not null default now(), -- время последнего изменения банера
    last_version bigint      not null default 1
);


CREATE TABLE IF NOT EXISTS features_tags_banner
(
    id         bigserial not null primary key,
    banner_id  bigint    not null references banner (id) on delete cascade,
    tag_id     bigint    not null,
    feature_id bigint    not null,
    deleted    boolean   not null default false
);

CREATE UNIQUE INDEX banner_identifier ON features_tags_banner (tag_id, feature_id) WHERE not deleted;

CREATE TABLE IF NOT EXISTS version_banner
(
    id         bigserial   not null primary key,
    version    bigint      not null default 1,
    banner_id  bigint      not null references banner (id) on delete cascade,
    content    jsonb       not null,
    created_at timestamptz not null default now(), -- время создания версии банера
    constraint banner_version UNIQUE (version, banner_id)
);

-- Для обновления поля update_at после обновление таблицы banner
CREATE OR REPLACE FUNCTION banner_update_trigger() RETURNS TRIGGER AS
$$
BEGIN
    UPDATE banner SET updated_at = now() WHERE id = NEW.id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER banner_update
    AFTER UPDATE
    ON banner
    FOR EACH ROW
    WHEN (OLD.* IS DISTINCT FROM NEW.*)
EXECUTE FUNCTION banner_update_trigger();

-- Для обновления поля update_at после обновление таблицы banner
CREATE OR REPLACE FUNCTION banner_insert_version_trigger() RETURNS TRIGGER AS
$$
DECLARE
    v_count      int    := 0;
    last_version bigint := 0;
BEGIN
    SELECT count(*) INTO v_count FROM version_banner WHERE banner_id = NEW.banner_id;
    IF v_count != 0 THEN
        SELECT max(version) INTO last_version FROM version_banner WHERE banner_id = NEW.banner_id;
        NEW.version = last_version + 1;
        DELETE FROM version_banner WHERE banner_id = NEW.banner_id and version < last_version - 1;
        UPDATE banner SET last_version = NEW.version WHERE id = NEW.banner_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER banner_insert_version
    BEFORE INSERT
    ON version_banner
    FOR EACH ROW
EXECUTE FUNCTION banner_insert_version_trigger();

-- Получены в ходе тестирование под нагрузкой запросов на получение
CREATE INDEX banner_feature on features_tags_banner (feature_id) WHERE not deleted;
CREATE INDEX banner_tag on features_tags_banner (tag_id) WHERE not deleted;
CREATE INDEX banner_feature_ids on features_tags_banner (banner_id) WHERE not deleted;
CREATE INDEX version_banner_id ON version_banner(banner_id);
CREATE INDEX feature_banner on features_tags_banner(banner_id, feature_id);
