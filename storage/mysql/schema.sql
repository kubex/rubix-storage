create table if not exists rubix.workspace_memberships
(
    user      varchar(64) null,
    workspace varchar(64) null,
    constraint user_workspace
    unique (user, workspace)
    );

create index user_index
    on rubix.workspace_memberships (user);

create index workspace_index
    on rubix.workspace_memberships (workspace);

create table if not exists rubix.workspaces
(
    uuid                  varchar(64)  not null
    primary key,
    name                  varchar(50)  null,
    alias                 varchar(50)  null,
    domain                varchar(120) null,
    installedApplications text         null
    );

create index workspaces_alias
    on rubix.workspaces (alias);

