create table workspace_memberships (
    user      varchar(64),
    workspace varchar(64),
    constraint user_workspace
        unique (user, workspace)
);

create index user_index
    on workspace_memberships(user);

create index workspace_index
    on workspace_memberships(workspace);

