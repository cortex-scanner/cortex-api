create table if not exists scans (
    id uuid primary key,
    scan_config_id uuid not null references scan_configs(id) on delete cascade,
    scan_start_time timestamptz,
    scan_end_time timestamptz,
    status varchar(255)
);