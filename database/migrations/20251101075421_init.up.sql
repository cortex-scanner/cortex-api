create table if not exists assets (
    id uuid primary key ,
    endpoint varchar(2048) not null unique
);

create table if not exists scan_configs (
    id uuid primary key,
    name varchar(1000) not null unique
);

create table if not exists scan_config_asset_map (
    scan_config_id uuid not null references scan_configs(id),
    asset_id uuid not null references assets(id),
    primary key (scan_config_id, asset_id)
)