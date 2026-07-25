create table lgas (
    id int generated always as indentity primary key,
    code varchar(10) not null unique,
    name varchar(100) not null unique,
    created_at timestamp not null default current_timestamp
);