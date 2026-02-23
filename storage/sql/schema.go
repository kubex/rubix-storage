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

	queries = append(queries, migQuery("CREATE TABLE `teams` ("+
		"`workspace`     varchar(64) NOT NULL,"+
		"`team`          varchar(64) NOT NULL,"+
		"`name`          varchar(64) NOT NULL,"+
		"`description` varchar(255) default '' not null,"+
		"PRIMARY KEY (`workspace`, `team`)"+
		");"))

	queries = append(queries, migQuery("CREATE TABLE `user_teams` ("+
		"`workspace` varchar(64) NOT NULL,"+
		"`user`      varchar(64) NOT NULL,"+
		"`team`      varchar(64) NOT NULL,"+
		"`level`      varchar(64) NOT NULL,"+ // member, manager, owner
		"PRIMARY KEY (`workspace`, `user`, `team`)"+
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

	queries = append(queries, migQuery("alter table `roles` ADD `lastUpdate` datetime default '';"))
	queries = append(queries, migQuery("alter table `workspace_memberships` ADD `lastUpdate` datetime default '';"))

	queries = append(queries, migQuery("CREATE TABLE `role_resources` ("+
		"`workspace`     varchar(64) NOT NULL,"+
		"`role`    			 varchar(64) NOT NULL,"+
		"`resource` 	   varchar(64) NOT NULL,"+
		"`resource_type` varchar(20) NOT NULL,"+ // brand,channel,department
		"PRIMARY KEY (`workspace`, `role`, `resource`)"+
		");"))

	queries = append(queries, migQuery("alter table `channels` ADD `maxLevel` int default 0;"))

	queries = append(queries, migQuery("CREATE TABLE `settings` ("+
		"`workspace` varchar(64) NOT NULL,"+
		"`vendor`    varchar(64) NOT NULL,"+
		"`app`       varchar(64) NULL,"+
		"`key`       varchar(64) NOT NULL,"+
		"`value`     text        NOT NULL,"+ // json
		"PRIMARY KEY (`workspace`, `vendor`, `app`, `key`)"+
		");"))

	queries = append(queries, migQuery("alter table `workspaces` ADD `oidcProvider` text null;"))

	queries = append(queries, migQuery("alter table `workspaces` ADD `emailDomainWhitelist` text null;"))

	queries = append(queries, migQuery("CREATE TABLE `distributors` ("+
		"`workspace`     varchar(64) NOT NULL,"+
		"`distributor`   varchar(64) NOT NULL,"+
		"`name`          varchar(64) NOT NULL,"+
		"`description` varchar(255) default '' not null,"+
		"PRIMARY KEY (`workspace`, `distributor`)"+
		");"))

	queries = append(queries, migQuery("CREATE TABLE `bpos` ("+
		"`workspace`     varchar(64) NOT NULL,"+
		"`bpo`           varchar(64) NOT NULL,"+
		"`name`          varchar(64) NOT NULL,"+
		"`description` varchar(255) default '' not null,"+
		"PRIMARY KEY (`workspace`, `bpo`)"+
		");"))

	queries = append(queries, migQuery("alter table `distributors` ADD `website_url` varchar(255) default '' not null;"))
	queries = append(queries, migQuery("alter table `distributors` ADD `logo_url` varchar(255) default '' not null;"))
	queries = append(queries, migQuery("alter table `bpos` ADD `website_url` varchar(255) default '' not null;"))
	queries = append(queries, migQuery("alter table `bpos` ADD `logo_url` varchar(255) default '' not null;"))

	queries = append(queries, migQuery("CREATE TABLE IF NOT EXISTS `bpo_managers` ("+
		"`workspace` varchar(64) NOT NULL,"+
		"`bpo`       varchar(64) NOT NULL,"+
		"`user`      varchar(64) NOT NULL,"+
		"PRIMARY KEY (`workspace`, `bpo`, `user`)"+
		");"))

	queries = append(queries, migQuery("CREATE TABLE IF NOT EXISTS `bpo_teams` ("+
		"`workspace` varchar(64) NOT NULL,"+
		"`bpo`       varchar(64) NOT NULL,"+
		"`team`      varchar(64) NOT NULL,"+
		"PRIMARY KEY (`workspace`, `bpo`, `team`)"+
		");"))

	queries = append(queries, migQuery("CREATE TABLE IF NOT EXISTS `bpo_roles` ("+
		"`workspace` varchar(64) NOT NULL,"+
		"`bpo`       varchar(64) NOT NULL,"+
		"`role`      varchar(64) NOT NULL,"+
		"PRIMARY KEY (`workspace`, `bpo`, `role`)"+
		");"))

	queries = append(queries, migQuery("CREATE TABLE `workspace_oidc_providers` ("+
		"`uuid`           varchar(64)  NOT NULL,"+
		"`workspace`      varchar(64)  NOT NULL,"+
		"`providerName`   varchar(120) NOT NULL,"+
		"`displayName`    varchar(255) NOT NULL DEFAULT '',"+
		"`clientID`       varchar(255) NOT NULL,"+
		"`clientSecret`   varchar(255) NULL,"+
		"`clientKeys`     text NULL,"+
		"`issuerURL`      varchar(255) NOT NULL,"+
		"PRIMARY KEY (`uuid`)"+
		");"))
	queries = append(queries, migQuery("CREATE INDEX `oidc_workspace` ON `workspace_oidc_providers`(`workspace`);"))

	queries = append(queries, migQuery("ALTER TABLE `workspace_oidc_providers` ADD `bpoID` varchar(64) NOT NULL DEFAULT '';"))

	queries = append(queries, migQuery("ALTER TABLE `workspace_oidc_providers` ADD `scimEnabled` tinyint(1) NOT NULL DEFAULT 0;"))
	queries = append(queries, migQuery("ALTER TABLE `workspace_oidc_providers` ADD `scimBearerToken` varchar(255) NOT NULL DEFAULT '';"))

	queries = append(queries, migQuery("ALTER TABLE `workspace_oidc_providers` ADD `scimSyncTeams` tinyint(1) NOT NULL DEFAULT 0;"))
	queries = append(queries, migQuery("ALTER TABLE `workspace_oidc_providers` ADD `scimSyncRoles` tinyint(1) NOT NULL DEFAULT 0;"))
	queries = append(queries, migQuery("ALTER TABLE `workspace_oidc_providers` ADD `scimAutoCreate` tinyint(1) NOT NULL DEFAULT 0;"))
	queries = append(queries, migQuery("ALTER TABLE `workspace_oidc_providers` ADD `scimDefaultGroupType` varchar(20) NOT NULL DEFAULT 'team';"))

	// SCIM Activity Log
	queries = append(queries, migQuery("CREATE TABLE `scim_activity_log` ("+
		"`id`           varchar(64)  NOT NULL,"+
		"`providerUUID` varchar(64)  NOT NULL,"+
		"`workspace`    varchar(64)  NOT NULL,"+
		"`timestamp`    datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,"+
		"`operation`    varchar(50)  NOT NULL,"+
		"`resource`     varchar(50)  NOT NULL,"+
		"`resourceID`   varchar(255) NOT NULL DEFAULT '',"+
		"`status`       varchar(20)  NOT NULL,"+
		"`detail`       text         NULL,"+
		"PRIMARY KEY (`id`)"+
		");"))
	queries = append(queries, migQuery("CREATE INDEX `scim_log_provider` ON `scim_activity_log`(`providerUUID`);"))
	queries = append(queries, migQuery("CREATE INDEX `scim_log_workspace` ON `scim_activity_log`(`workspace`);"))

	// Workspace Users (OIDC directory)
	queries = append(queries, migQuery("CREATE TABLE `workspace_users` ("+
		"`user_id`        varchar(64)  NOT NULL,"+
		"`workspace`      varchar(64)  NOT NULL,"+
		"`name`           varchar(64)  NULL,"+
		"`email`          varchar(128) NULL,"+
		"`oidc_provider`  varchar(64)  NOT NULL,"+
		"`scim_managed`   tinyint(1)   NOT NULL DEFAULT 0,"+
		"`auto_created`   tinyint(1)   NOT NULL DEFAULT 0,"+
		"`last_sync_time` datetime     NULL,"+
		"`created_at`     datetime     NOT NULL DEFAULT CURRENT_TIMESTAMP,"+
		"PRIMARY KEY (`user_id`)"+
		");"))
	queries = append(queries, migQuery("CREATE INDEX `wu_workspace` ON `workspace_users`(`workspace`);"))
	queries = append(queries, migQuery("CREATE INDEX `wu_provider` ON `workspace_users`(`oidc_provider`);"))
	queries = append(queries, migQuery("CREATE INDEX `wu_workspace_provider` ON `workspace_users`(`workspace`, `oidc_provider`);"))

	queries = append(queries, migQuery("ALTER TABLE `teams` ADD `scimManaged` tinyint(1) NOT NULL DEFAULT 0;"))
	queries = append(queries, migQuery("ALTER TABLE `roles` ADD `scimManaged` tinyint(1) NOT NULL DEFAULT 0;"))

	// Member approval queue
	queries = append(queries, migQuery("ALTER TABLE `workspace_memberships` ADD `source` varchar(20) NOT NULL DEFAULT '';"))
	queries = append(queries, migQuery("ALTER TABLE `workspaces` ADD `memberApprovalMode` varchar(10) NOT NULL DEFAULT 'auto';"))
	queries = append(queries, migQuery("ALTER TABLE `workspace_oidc_providers` ADD `autoAcceptMembers` tinyint(1) NOT NULL DEFAULT 0;"))
	queries = append(queries, migQuery("ALTER TABLE `workspaces` ADD `emailDomainApproval` text;"))

	// IP Groups
	queries = append(queries, migQuery("CREATE TABLE `ip_groups` ("+
		"`workspace`    varchar(64)  NOT NULL,"+
		"`ip_group`     varchar(64)  NOT NULL,"+
		"`name`         varchar(64)  NOT NULL,"+
		"`description`  varchar(255) NOT NULL DEFAULT '',"+
		"`source`       varchar(20)  NOT NULL DEFAULT 'manual',"+
		"`entries`      text         NULL,"+
		"`externalUrl`  varchar(512) NOT NULL DEFAULT '',"+
		"`jsonPath`     varchar(128) NOT NULL DEFAULT '',"+
		"`lastSynced`   datetime     NULL,"+
		"`entryCount`   int          NOT NULL DEFAULT 0,"+
		"PRIMARY KEY (`workspace`, `ip_group`)"+
		");"))

	queries = append(queries, migQuery("ALTER TABLE `workspace_oidc_providers` ADD `assumeMFA` tinyint(1) NOT NULL DEFAULT 0;"))
	queries = append(queries, migQuery("ALTER TABLE `workspace_oidc_providers` ADD `assumeVerified` tinyint(1) NOT NULL DEFAULT 0;"))
	queries = append(queries, migQuery("ALTER TABLE `workspace_oidc_providers` ADD `maxSessionAge` int NOT NULL DEFAULT 0;"))

	return queries
}
