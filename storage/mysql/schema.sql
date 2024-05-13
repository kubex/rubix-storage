create table rubix.workspace_memberships (
    user      varchar(64) null,
    workspace varchar(64) null,
    role      varchar(20) null,
    constraint user_workspace unique (user, workspace)
);

create index user_index on rubix.workspace_memberships(user);
create index workspace_index on rubix.workspace_memberships(workspace);

create table rubix.workspaces (
    uuid                  varchar(64) not null primary key,
    name                  varchar(50) null,
    alias                 varchar(50) null,
    domain                varchar(120) null,
    installedApplications text null
);

create index workspaces_alias on rubix.workspaces(alias);


create table rubix.auth_data (
    workspace varchar(64) not null,
    user      varchar(64) null,
    vendor    varchar(64) not null,
    app       varchar(64) null,
    `key`     varchar(64) not null,
    `value`   text        not null
);

create unique index `wuvak` on rubix.auth_data(workspace, user, vendor, app, `key`);

create table rubix.users (
    user varchar(64) null,
    name varchar(64) not null
);

CREATE TABLE rubix.`roles` (
    `workspace` varchar(64) NOT NULL,
    `role`      varchar(64) NOT NULL,
    `name`      varchar(64) NOT NULL,
    PRIMARY KEY (`workspace`, `role`)
)

CREATE TABLE rubix.`role_permissions` (
    `workspace`  varchar(64)  NOT NULL,
    `role`       varchar(64)  NOT NULL,
    `permission` varchar(255) NOT NULL,
    `resource`   varchar(255) NOT NULL,
    `allow`      tinyint(1) NOT NULL,
    PRIMARY KEY (`workspace`, `role`, `permission`, `resource`)
)

CREATE TABLE rubix.`user_roles` (
    `workspace` varchar(64) NOT NULL,
    `user`      varchar(64) NOT NULL,
    `role`      varchar(64) NOT NULL,
    PRIMARY KEY (`workspace`, `user`, `role`)
)

CREATE TABLE rubix.`user_status` (
    `workspace`     varchar(64) NOT NULL,
    `user`          varchar(64) NOT NULL,
    `state`         varchar(10) NOT NULL,
    `extendedState` varchar(50) NOT NULL,
    `expiry`        datetime             DEFAULT NULL,
    `applied`       datetime    NOT NULL,
    `id`            varchar(64) NOT NULL DEFAULT '',
    `afterId`       varchar(64)          DEFAULT NULL,
    `duration`      int(11)     NOT NULL DEFAULT 0,
    `clearOnLogout` tinyint(1)  NOT NULL DEFAULT 0,
    PRIMARY KEY (`workspace`, `user`, `id`)
)