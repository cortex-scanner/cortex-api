create table if not exists asset_history (
    id uuid primary key,
    asset_id uuid references assets(id) on delete cascade,
    event_type varchar(64),
    user_id uuid references users(id),
    timestamp timestamptz not null default now(),
    event_data jsonb
);
