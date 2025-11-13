drop table if exists scan_config_asset_map;

alter table scan_configs add column type varchar(255);
alter table scan_configs add column engine varchar(255);
alter table scans drop column type;

create table if not exists scan_asset_map (
    scan_id uuid not null references scans(id),
    asset_id uuid not null references assets(id),
    primary key (scan_id, asset_id)
);

-- insert default naabu scanner config
insert into scan_configs (id, name, type, engine) values ('5ce58a6d-1c85-4e6f-9dda-2464fb5dc602', 'Naabu Default', 'discovery', 'naabu');