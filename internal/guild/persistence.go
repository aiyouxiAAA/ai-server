package guild

import (
	"fmt"
)

func (service *Service) initSchema() error {
	if service.db == nil {
		return nil
	}
	queries := []string{
		`CREATE TABLE IF NOT EXISTS classic_guilds (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			logo_id TEXT NOT NULL,
			notice TEXT NOT NULL,
			creator_role_id TEXT NOT NULL,
			creator_name TEXT NOT NULL,
			max_member INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS classic_guild_members (
			role_id TEXT PRIMARY KEY,
			guild_id TEXT NOT NULL,
			role_name TEXT NOT NULL,
			level INTEGER NOT NULL,
			position TEXT NOT NULL,
			auth INTEGER NOT NULL,
			FOREIGN KEY(guild_id) REFERENCES classic_guilds(id) ON DELETE CASCADE
		)`,
	}
	for _, query := range queries {
		if _, err := service.db.Exec(query); err != nil {
			return fmt.Errorf("initialize guild sqlite schema: %w", err)
		}
	}
	return nil
}

func (service *Service) load() error {
	if service.db == nil {
		return nil
	}
	guildRows, err := service.db.Query(`
		SELECT id, name, logo_id, notice, creator_role_id, creator_name, max_member
		FROM classic_guilds
		ORDER BY id ASC
	`)
	if err != nil {
		return fmt.Errorf("query guilds: %w", err)
	}
	defer guildRows.Close()
	for guildRows.Next() {
		var guild Guild
		if err := guildRows.Scan(
			&guild.ID,
			&guild.Name,
			&guild.LogoID,
			&guild.Notice,
			&guild.CreatorRoleID,
			&guild.CreatorName,
			&guild.MaxMember,
		); err != nil {
			return fmt.Errorf("scan guild: %w", err)
		}
		service.guilds[guild.ID] = guild
		var seq int
		if _, err := fmt.Sscanf(guild.ID, "guild-%04d", &seq); err == nil && seq > service.nextSeq {
			service.nextSeq = seq
		}
	}
	if err := guildRows.Err(); err != nil {
		return fmt.Errorf("iterate guilds: %w", err)
	}

	memberRows, err := service.db.Query(`
		SELECT role_id, guild_id, role_name, level, position, auth
		FROM classic_guild_members
		ORDER BY guild_id ASC, auth DESC, role_name ASC
	`)
	if err != nil {
		return fmt.Errorf("query guild members: %w", err)
	}
	defer memberRows.Close()
	for memberRows.Next() {
		var member Member
		if err := memberRows.Scan(
			&member.RoleID,
			&member.GuildID,
			&member.RoleName,
			&member.Level,
			&member.Position,
			&member.Auth,
		); err != nil {
			return fmt.Errorf("scan guild member: %w", err)
		}
		member.Online = false
		service.members[member.RoleID] = member
	}
	if err := memberRows.Err(); err != nil {
		return fmt.Errorf("iterate guild members: %w", err)
	}
	return nil
}

func (service *Service) persistGuildLocked(guild Guild, members []Member) error {
	if service.db == nil {
		return nil
	}
	tx, err := service.db.Begin()
	if err != nil {
		return fmt.Errorf("begin guild transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	if _, err = tx.Exec(
		`INSERT INTO classic_guilds (id, name, logo_id, notice, creator_role_id, creator_name, max_member)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   name = excluded.name,
		   logo_id = excluded.logo_id,
		   notice = excluded.notice,
		   creator_role_id = excluded.creator_role_id,
		   creator_name = excluded.creator_name,
		   max_member = excluded.max_member`,
		guild.ID,
		guild.Name,
		guild.LogoID,
		guild.Notice,
		guild.CreatorRoleID,
		guild.CreatorName,
		guild.MaxMember,
	); err != nil {
		return fmt.Errorf("upsert guild %s: %w", guild.ID, err)
	}
	if _, err = tx.Exec(`DELETE FROM classic_guild_members WHERE guild_id = ?`, guild.ID); err != nil {
		return fmt.Errorf("reset guild members %s: %w", guild.ID, err)
	}
	for _, member := range members {
		if _, err = tx.Exec(
			`INSERT INTO classic_guild_members (role_id, guild_id, role_name, level, position, auth)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			member.RoleID,
			guild.ID,
			member.RoleName,
			member.Level,
			member.Position,
			member.Auth,
		); err != nil {
			return fmt.Errorf("insert guild member %s: %w", member.RoleID, err)
		}
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit guild transaction: %w", err)
	}
	return nil
}

func (service *Service) deleteGuildLocked(guildID string) error {
	if service.db == nil {
		return nil
	}
	if _, err := service.db.Exec(`DELETE FROM classic_guild_members WHERE guild_id = ?`, guildID); err != nil {
		return fmt.Errorf("delete guild members %s: %w", guildID, err)
	}
	if _, err := service.db.Exec(`DELETE FROM classic_guilds WHERE id = ?`, guildID); err != nil {
		return fmt.Errorf("delete guild %s: %w", guildID, err)
	}
	return nil
}
