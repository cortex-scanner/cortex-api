create table if not exists asset_findings (
    id uuid primary key,
    asset_id uuid references assets(id) on delete cascade,
    first_seen timestamptz not null,
    last_seen timestamptz not null,
    type varchar(255) not null,
    data jsonb
);

drop table asset_discovery;