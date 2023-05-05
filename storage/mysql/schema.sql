create table if not exists rubix.workspace_memberships
(
    user      varchar(64) null,
    workspace varchar(64) null,
    role       varchar(20) null,
    constraint user_workspace unique (user, workspace)
    );

create index user_index on rubix.workspace_memberships (user);
create index workspace_index on rubix.workspace_memberships (workspace);

create table if not exists rubix.workspaces
(
    uuid                  varchar(64)  not null primary key,
    name                  varchar(50)  null,
    alias                 varchar(50)  null,
    domain                varchar(120) null,
    installedApplications text         null
);

create index workspaces_alias on rubix.workspaces (alias);


create table if not exists rubix.auth_data (
    workspace varchar(64) not null,
    user      varchar(64) not null,
    vendor    varchar(64) not null,
    app       varchar(64) null,
    `key`     varchar(64) not null,
    `value`     text        not null
);

create unique index `wuvak` on rubix.auth_data(workspace, user, vendor, app, `key`);

