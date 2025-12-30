package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/marcelom97/scimgateway/scim"
	_ "modernc.org/sqlite"
)

// SQLitePlugin implements a SQLite-backed SCIM plugin
type SQLitePlugin struct {
	name string
	db   *sqlx.DB
}

// UserData wraps scim.User and implements sql.Scanner and driver.Valuer
type UserData struct {
	User *scim.User
}

// Scan implements sql.Scanner interface for reading from database
func (u *UserData) Scan(value any) error {
	if value == nil {
		u.User = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("failed to scan UserData: expected []byte or string, got %T", value)
		}
		bytes = []byte(str)
	}

	u.User = &scim.User{}
	if err := json.Unmarshal(bytes, u.User); err != nil {
		return fmt.Errorf("failed to unmarshal user data: %w", err)
	}

	return nil
}

// Value implements driver.Valuer interface for writing to database
func (u UserData) Value() (driver.Value, error) {
	if u.User == nil {
		return nil, nil
	}

	bytes, err := json.Marshal(u.User)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal user data: %w", err)
	}

	return string(bytes), nil
}

// GroupData wraps scim.Group and implements sql.Scanner and driver.Valuer
type GroupData struct {
	Group *scim.Group
}

// Scan implements sql.Scanner interface for reading from database
func (g *GroupData) Scan(value any) error {
	if value == nil {
		g.Group = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("failed to scan GroupData: expected []byte or string, got %T", value)
		}
		bytes = []byte(str)
	}

	g.Group = &scim.Group{}
	if err := json.Unmarshal(bytes, g.Group); err != nil {
		return fmt.Errorf("failed to unmarshal group data: %w", err)
	}

	return nil
}

// Value implements driver.Valuer interface for writing to database
func (g GroupData) Value() (driver.Value, error) {
	if g.Group == nil {
		return nil, nil
	}

	bytes, err := json.Marshal(g.Group)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal group data: %w", err)
	}

	return string(bytes), nil
}

// userRow represents a user row in the database
type userRow struct {
	ID        string    `db:"id"`
	Username  string    `db:"username"`
	Data      UserData  `db:"data"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// groupRow represents a group row in the database
type groupRow struct {
	ID          string    `db:"id"`
	DisplayName string    `db:"display_name"`
	Data        GroupData `db:"data"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

// NewSQLitePlugin creates a new SQLite plugin
func NewSQLitePlugin(name string, dbPath string) (*SQLitePlugin, error) {
	db, err := sqlx.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	plugin := &SQLitePlugin{
		name: name,
		db:   db,
	}

	// Initialize database schema
	if err := plugin.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return plugin, nil
}

// initSchema creates the database tables if they don't exist
func (p *SQLitePlugin) initSchema() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			username TEXT NOT NULL,
			data TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_users_username ON users(username)`,
		`CREATE TABLE IF NOT EXISTS groups (
			id TEXT PRIMARY KEY,
			display_name TEXT NOT NULL,
			data TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_groups_display_name ON groups(display_name)`,
	}

	for _, query := range queries {
		if _, err := p.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute schema query: %w", err)
		}
	}

	return nil
}

// Name returns the plugin name
func (p *SQLitePlugin) Name() string {
	return p.name
}

// Close closes the database connection
func (p *SQLitePlugin) Close() error {
	return p.db.Close()
}

// GetUsers retrieves all users
func (p *SQLitePlugin) GetUsers(ctx context.Context, params scim.QueryParams) ([]*scim.User, error) {
	var rows []userRow
	query := `SELECT id, username, data, created_at, updated_at FROM users`

	if err := p.db.SelectContext(ctx, &rows, query); err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}

	users := make([]*scim.User, 0, len(rows))
	for _, row := range rows {
		if row.Data.User != nil {
			users = append(users, row.Data.User)
		}
	}

	return users, nil
}

// CreateUser creates a new user
func (p *SQLitePlugin) CreateUser(ctx context.Context, user *scim.User) (*scim.User, error) {
	// Generate ID if not provided
	if user.ID == "" {
		user.ID = uuid.New().String()
	}

	// Set schemas if not provided
	if len(user.Schemas) == 0 {
		user.Schemas = []string{scim.SchemaUser}
	}

	var exists bool
	// Check for existing username
	err := p.db.GetContext(ctx, &exists, "SELECT EXISTS(SELECT 1 FROM users WHERE username = ?)", user.UserName)
	if err != nil {
		return nil, scim.ErrInternalServer(fmt.Sprintf("failed to check existing username: %v", err))
	}

	if exists {
		return nil, scim.ErrUniqueness(
			fmt.Sprintf("userName '%s' already exists", user.UserName),
		)
	}

	// Set meta
	now := time.Now()
	user.Meta = &scim.Meta{
		ResourceType: "User",
		Created:      &now,
		LastModified: &now,
		Version:      fmt.Sprintf("W/\"%s\"", user.ID),
	}

	// Insert user into database
	query := `INSERT INTO users (id, username, data, created_at, updated_at) VALUES (:id, :username, :data, :created_at, :updated_at)`

	row := userRow{
		ID:        user.ID,
		Username:  user.UserName,
		Data:      UserData{User: user},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if _, err := p.db.NamedExecContext(ctx, query, row); err != nil {
		return nil, scim.ErrInternalServer(fmt.Sprintf("failed to insert user: %v", err))
	}

	return user, nil
}

// GetUser retrieves a specific user by ID
func (p *SQLitePlugin) GetUser(ctx context.Context, id string, attributes []string) (*scim.User, error) {
	var row userRow
	query := `SELECT id, username, data, created_at, updated_at FROM users WHERE id = ?`

	if err := p.db.GetContext(ctx, &row, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, scim.ErrNotFound("User", id)
		}
		return nil, scim.ErrInternalServer(fmt.Sprintf("failed to get user: %v", err))
	}

	return row.Data.User, nil
}

// ModifyUser updates a user's attributes
func (p *SQLitePlugin) ModifyUser(ctx context.Context, id string, patch *scim.PatchOp) error {
	// Get existing user (returns ErrNotFound if not exists)
	user, err := p.GetUser(ctx, id, nil)
	if err != nil {
		return err
	}

	// Apply patch operations
	patcher := scim.NewPatchProcessor()
	if err := patcher.ApplyPatch(user, patch); err != nil {
		return scim.ErrInvalidSyntax(fmt.Sprintf("failed to apply patch: %v", err))
	}

	// Update metadata
	now := time.Now()
	user.Meta.LastModified = &now
	user.Meta.Version = fmt.Sprintf("W/\"%s-%d\"", id, now.Unix())

	// Update user in database
	query := `UPDATE users SET username = :username, data = :data, updated_at = :updated_at WHERE id = :id`

	row := userRow{
		ID:        user.ID,
		Username:  user.UserName,
		Data:      UserData{User: user},
		UpdatedAt: now,
	}

	if _, err := p.db.NamedExecContext(ctx, query, row); err != nil {
		return scim.ErrInternalServer(fmt.Sprintf("failed to update user: %v", err))
	}

	return nil
}

// DeleteUser deletes a user
func (p *SQLitePlugin) DeleteUser(ctx context.Context, id string) error {
	result, err := p.db.ExecContext(ctx, "DELETE FROM users WHERE id = ?", id)
	if err != nil {
		return scim.ErrInternalServer(fmt.Sprintf("failed to delete user: %v", err))
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return scim.ErrInternalServer(fmt.Sprintf("failed to get rows affected: %v", err))
	}

	if rows == 0 {
		return scim.ErrNotFound("User", id)
	}

	return nil
}

// GetGroups retrieves all groups
func (p *SQLitePlugin) GetGroups(ctx context.Context, params scim.QueryParams) ([]*scim.Group, error) {
	var rows []groupRow
	query := `SELECT id, display_name, data, created_at, updated_at FROM groups`

	if err := p.db.SelectContext(ctx, &rows, query); err != nil {
		return nil, fmt.Errorf("failed to query groups: %w", err)
	}

	groups := make([]*scim.Group, 0, len(rows))
	for _, row := range rows {
		if row.Data.Group != nil {
			groups = append(groups, row.Data.Group)
		}
	}

	return groups, nil
}

// CreateGroup creates a new group
func (p *SQLitePlugin) CreateGroup(ctx context.Context, group *scim.Group) (*scim.Group, error) {
	// Generate ID if not provided
	if group.ID == "" {
		group.ID = uuid.New().String()
	}

	// Set schemas if not provided
	if len(group.Schemas) == 0 {
		group.Schemas = []string{scim.SchemaGroup}
	}

	var exists bool
	// Check for existing displayName
	err := p.db.GetContext(ctx, &exists, "SELECT EXISTS(SELECT 1 FROM groups WHERE display_name = ?)", group.DisplayName)
	if err != nil {
		return nil, scim.ErrInternalServer(fmt.Sprintf("failed to check existing displayName: %v", err))
	}

	if exists {
		return nil, scim.ErrUniqueness(
			fmt.Sprintf("displayName '%s' already exists", group.DisplayName),
		)
	}

	// Set meta
	now := time.Now()
	group.Meta = &scim.Meta{
		ResourceType: "Group",
		Created:      &now,
		LastModified: &now,
		Version:      fmt.Sprintf("W/\"%s\"", group.ID),
	}

	// Insert group into database
	query := `INSERT INTO groups (id, display_name, data, created_at, updated_at) VALUES (:id, :display_name, :data, :created_at, :updated_at)`

	row := groupRow{
		ID:          group.ID,
		DisplayName: group.DisplayName,
		Data:        GroupData{Group: group},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if _, err := p.db.NamedExecContext(ctx, query, row); err != nil {
		return nil, scim.ErrInternalServer(fmt.Sprintf("failed to insert group: %v", err))
	}

	return group, nil
}

// GetGroup retrieves a specific group by ID
func (p *SQLitePlugin) GetGroup(ctx context.Context, id string, attributes []string) (*scim.Group, error) {
	var row groupRow
	query := `SELECT id, display_name, data, created_at, updated_at FROM groups WHERE id = ?`

	if err := p.db.GetContext(ctx, &row, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, scim.ErrNotFound("Group", id)
		}
		return nil, scim.ErrInternalServer(fmt.Sprintf("failed to get group: %v", err))
	}

	return row.Data.Group, nil
}

// ModifyGroup updates a group's attributes
func (p *SQLitePlugin) ModifyGroup(ctx context.Context, id string, patch *scim.PatchOp) error {
	// Get existing group (returns ErrNotFound if not exists)
	group, err := p.GetGroup(ctx, id, nil)
	if err != nil {
		return err
	}

	// Apply patch operations
	patcher := scim.NewPatchProcessor()
	if err := patcher.ApplyPatch(group, patch); err != nil {
		return scim.ErrInvalidSyntax(fmt.Sprintf("failed to apply patch: %v", err))
	}

	// Update metadata
	now := time.Now()
	group.Meta.LastModified = &now
	group.Meta.Version = fmt.Sprintf("W/\"%s-%d\"", id, now.Unix())

	// Update group in database
	query := `UPDATE groups SET display_name = :display_name, data = :data, updated_at = :updated_at WHERE id = :id`

	row := groupRow{
		ID:          group.ID,
		DisplayName: group.DisplayName,
		Data:        GroupData{Group: group},
		UpdatedAt:   now,
	}

	if _, err := p.db.NamedExecContext(ctx, query, row); err != nil {
		return scim.ErrInternalServer(fmt.Sprintf("failed to update group: %v", err))
	}

	return nil
}

// DeleteGroup deletes a group
func (p *SQLitePlugin) DeleteGroup(ctx context.Context, id string) error {
	result, err := p.db.ExecContext(ctx, "DELETE FROM groups WHERE id = ?", id)
	if err != nil {
		return scim.ErrInternalServer(fmt.Sprintf("failed to delete group: %v", err))
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return scim.ErrInternalServer(fmt.Sprintf("failed to get rows affected: %v", err))
	}

	if rows == 0 {
		return scim.ErrNotFound("Group", id)
	}

	return nil
}
