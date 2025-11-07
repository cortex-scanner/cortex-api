create table if not exists users (
    id uuid primary key,
    provider varchar(32) not null,
    username varchar(255) not null,
    email varchar(255),
    display_name varchar(255),
    password varchar(255),
    created_at timestamptz not null default now()
);

-- default admin user; password=admin
insert into users (id, provider, username, email, display_name, password) values ('354ce225-7a97-4daa-8255-5fef049e8b1d', 'local', 'admin', '', 'Administrator', '$argon2id$v=19$m=16,t=2,p=1$QXNjcVZUb3NuZGF0VDZJdg$WQ5019fGGqYymE6isgsOtg');

create table if not exists sessions (
    token varchar(255) primary key,
    user_id uuid not null references users(id) on delete cascade,
    created_at timestamptz not null default now(),
    expires_at timestamptz not null,
    source_ip varchar(64),
    revoked boolean not null default false,
    user_agent varchar(255)
);