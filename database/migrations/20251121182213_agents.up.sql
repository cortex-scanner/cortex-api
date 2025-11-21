create table if not exists agents (
    id varchar(16) primary key,
    name varchar(255) unique not null,
    auth_token_hash varchar(255) not null,
    created_at timestamptz not null default now()
);