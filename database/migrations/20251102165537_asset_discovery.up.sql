create table if not exists asset_discovery (
    asset_id uuid not null references assets(id) on delete cascade,
    port integer not null,
    protocol varchar(3),
    first_seen timestamptz,
    last_seen timestamptz,
    primary key (asset_id, port, protocol)
);

alter table scans add type varchar(32);