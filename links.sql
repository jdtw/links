create table if not exists links (
  path text primary key,
  link text not null,
  segments int not null
);