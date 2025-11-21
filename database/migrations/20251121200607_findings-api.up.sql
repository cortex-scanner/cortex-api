alter table asset_findings rename column first_seen to created_at;
alter table asset_findings drop column last_seen;
alter table asset_findings add column finding_hash varchar(64);
alter table asset_findings add column agent_id varchar(16) references agents(id) on delete cascade;