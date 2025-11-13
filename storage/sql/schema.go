package sql

import (
	"crypto/sha1"
	"fmt"
)

type migration struct {
	key   string
	query string
}

func migQuery(query string) migration {
	return migration{
		key:   fmt.Sprintf("%x", sha1.Sum([]byte(query)))[0:8],
		query: query,
	}
}

func migrations() []migration {
	var queries []migration

	// Memberships
	queries = append(queries, migQuery("create table workspace_memberships ("+
		"user        varchar(64)                           not null,"+
		"workspace   varchar(64)                           not null,"+
		"type        varchar(20) default ''                not null,"+
		"partner_id  varchar(64) default ''                not null,"+
		"since       datetime    default CURRENT_TIMESTAMP not null,"+
		"state       varchar(20) default 0                 not null,"+
		"state_since datetime    default CURRENT_TIMESTAMP not null,"+
		"PRIMARY KEY (`user`, `workspace`)"+
		");"))
	queries = append(queries, migQuery(`create index user_index on workspace_memberships(user);`))
	queries = append(queries, migQuery(`create index workspace_index on workspace_memberships(workspace);`))

	// Workspaces
	queries = append(queries, migQuery("create table workspaces ("+
		"uuid                  varchar(64) not null,"+
		"name                  varchar(50) null,"+
		"alias                 varchar(50) null,"+
		"domain                varchar(120) null,"+
		"icon                  varchar(255) null,"+
		"installedApplications text null,"+
		"defaultApp            varchar(120) null,"+
		"systemVendors         varchar(120) null,"+
		"footerParts           text null,"+
		"PRIMARY KEY (`uuid`)"+
		");"))
	queries = append(queries, migQuery(`create index workspaces_alias on workspaces(alias);`))

	// Auth Data
	queries = append(queries, migQuery("create table auth_data ("+
		"`workspace` varchar(64) not null,"+
		"`user`      varchar(64) null,"+
		"`vendor`    varchar(64) not null,"+
		"`app`       varchar(64) null,"+
		"`key`       varchar(64) not null,"+
		"`value`     text        not null,"+
		"PRIMARY KEY (`workspace`, `vendor`, `app`, `user`, `key`)"+
		");"))
	queries = append(queries, migQuery("create unique index `wuvak` on auth_data(`workspace`, `user`, `vendor`, `app`, `key`);"))

	// Users
	queries = append(queries, migQuery("create table users ("+
		"`user`  varchar(64) NOT NULL,"+
		"`name`  varchar(64) NOT NULL,"+
		"`email` varchar(128) DEFAULT NULL,"+
		"PRIMARY KEY (`user`)"+
		");"))

	// Roles
	queries = append(queries, migQuery("CREATE TABLE `roles` ("+
		"`workspace`   varchar(64)             NOT NULL,"+
		"`role`        varchar(64)             NOT NULL,"+
		"`name`        varchar(64)             NOT NULL,"+
		"`description` varchar(255) default '' not null,"+
		"PRIMARY KEY (`workspace`, `role`)"+
		");"))

	queries = append(queries, migQuery("CREATE TABLE `role_permissions` ("+
		"workspace  varchar(64)             NOT NULL,"+
		"role       varchar(64)             NOT NULL,"+
		"permission varchar(255)            NOT NULL,"+
		"resource   varchar(255) default '' NOT NULL,"+
		"allow      tinyint(1)   default 1  NOT NULL,"+
		"meta       varchar(255) default '' NOT NULL,"+
		"PRIMARY KEY (`workspace`, `role`, `permission`, `resource`)"+
		");"))
	queries = append(queries, migQuery("CREATE TABLE `user_roles` ("+
		"`workspace` varchar(64) NOT NULL,"+
		"`user`      varchar(64) NOT NULL,"+
		"`role`      varchar(64) NOT NULL,"+
		"PRIMARY KEY (`workspace`, `user`, `role`)"+
		");"))
	queries = append(queries, migQuery("create index role_users on `user_roles`(workspace, role);"))
	queries = append(queries, migQuery("CREATE TABLE `user_status` ("+
		"`workspace`     varchar(64) NOT NULL,"+
		"`user`          varchar(64) NOT NULL,"+
		"`state`         varchar(10) NOT NULL,"+
		"`extendedState` varchar(50) NOT NULL,"+
		"`expiry`        datetime             DEFAULT NULL,"+
		"`applied`       datetime    NOT NULL,"+
		"`id`            varchar(64) NOT NULL DEFAULT '',"+
		"`afterId`       varchar(64)          DEFAULT NULL,"+
		"`duration`      int(11)     NOT NULL DEFAULT 0,"+
		"`clearOnLogout` tinyint(1)  NOT NULL DEFAULT 0,"+
		"PRIMARY KEY (`workspace`, `user`, `id`)"+
		");"))

	queries = append(queries, migQuery("alter table `workspaces` "+
		"ADD `accessCondition` text null"+
		";"))

	queries = append(queries, migQuery("alter table `roles` "+
		"ADD `conditions` text null"+
		";"))

	queries = append(queries, migQuery("alter table `role_permissions` "+
		"ADD `options` text null"+
		";"))

	queries = append(queries, migQuery("CREATE TABLE `groups` ("+
		"`workspace`     varchar(64) NOT NULL,"+
		"`group`          varchar(64) NOT NULL,"+
		"`name`          varchar(64) NOT NULL,"+
		"`description` varchar(255) default '' not null,"+
		"PRIMARY KEY (`workspace`, `group`)"+
		");"))

	queries = append(queries, migQuery("CREATE TABLE `user_groups` ("+
		"`workspace` varchar(64) NOT NULL,"+
		"`user`      varchar(64) NOT NULL,"+
		"`group`      varchar(64) NOT NULL,"+
		"`level`      varchar(64) NOT NULL,"+ // member, manager, owner
		"PRIMARY KEY (`workspace`, `user`, `group`)"+
		");"))

	queries = append(queries, migQuery("CREATE TABLE `brands` ("+
		"`workspace`     varchar(64) NOT NULL,"+
		"`brand`          varchar(64) NOT NULL,"+
		"`name`          varchar(64) NOT NULL,"+
		"`description` varchar(255) default '' not null,"+
		"PRIMARY KEY (`workspace`, `brand`)"+
		");"))

	queries = append(queries, migQuery("CREATE TABLE `departments` ("+
		"`workspace`     varchar(64) NOT NULL,"+
		"`department`          varchar(64) NOT NULL,"+
		"`name`          varchar(64) NOT NULL,"+
		"`description` varchar(255) default '' not null,"+
		"PRIMARY KEY (`workspace`, `department`)"+
		");"))

	queries = append(queries, migQuery("CREATE TABLE `channels` ("+
		"`workspace`     varchar(64) NOT NULL,"+
		"`channel`          varchar(64) NOT NULL,"+
		"`department`          varchar(64) NOT NULL,"+
		"`name`          varchar(64) NOT NULL,"+
		"`description` varchar(255) default '' not null,"+
		"PRIMARY KEY (`workspace`, `channel`)"+
		");"))

	return queries
}
