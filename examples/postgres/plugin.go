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

	_ "github.com/lib/pq"
)

// PostgresPlugin implements a PostgreSQL-backed SCIM plugin
type PostgresPlugin struct {
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

// NewPostgresPlugin creates a new PostgreSQL plugin
func NewPostgresPlugin(name string, connStr string) (*PostgresPlugin, error) {
	db, err := sqlx.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(2)

	// Verify connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close() // nolint:errcheck
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	plugin := &PostgresPlugin{
		name: name,
		db:   db,
	}

	// Initialize database schema
	if err := plugin.initSchema(); err != nil {
		db.Close() // nolint:errcheck
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return plugin, nil
}

// initSchema creates the database tables if they don't exist
func (p *PostgresPlugin) initSchema() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			username TEXT NOT NULL,
			data JSONB NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_users_username ON users(username)`,
		// GIN index for efficient JSONB queries
		`CREATE INDEX IF NOT EXISTS idx_users_data ON users USING GIN(data)`,
		`CREATE TABLE IF NOT EXISTS groups (
			id TEXT PRIMARY KEY,
			display_name TEXT NOT NULL,
			data JSONB NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_groups_display_name ON groups(display_name)`,
		// GIN index for efficient JSONB queries
		`CREATE INDEX IF NOT EXISTS idx_groups_data ON groups USING GIN(data)`,
	}

	for _, query := range queries {
		if _, err := p.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute schema query: %w", err)
		}
	}

	return nil
}

// Name returns the plugin name
func (p *PostgresPlugin) Name() string {
	return p.name
}

// Close closes the database connection
func (p *PostgresPlugin) Close() error {
	return p.db.Close()
}

// GetUsers retrieves users with optional filtering, sorting, and pagination
func (p *PostgresPlugin) GetUsers(ctx context.Context, params scim.QueryParams) ([]*scim.User, error) {
	// Build optimized query from QueryParams
	qb := NewQueryBuilder("users", "data", UserAttributeMapping)
	query, args := qb.Build(params)

	// Rebind for PostgreSQL ($1, $2, etc.)
	query = p.db.Rebind(query)

	var rows []userRow
	if err := p.db.SelectContext(ctx, &rows, query, args...); err != nil {
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
func (p *PostgresPlugin) CreateUser(ctx context.Context, user *scim.User) (*scim.User, error) {
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
	err := p.db.GetContext(ctx, &exists, "SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)", user.UserName)
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
	query := `INSERT INTO users (id, username, data, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`

	userData := UserData{User: user}
	if _, err := p.db.ExecContext(ctx, query, user.ID, user.UserName, userData, now, now); err != nil {
		return nil, scim.ErrInternalServer(fmt.Sprintf("failed to insert user: %v", err))
	}

	return user, nil
}

// GetUser retrieves a specific user by ID
func (p *PostgresPlugin) GetUser(ctx context.Context, id string, attributes []string) (*scim.User, error) {
	var row userRow
	query := `SELECT id, username, data, created_at, updated_at FROM users WHERE id = $1`

	if err := p.db.GetContext(ctx, &row, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, scim.ErrNotFound("User", id)
		}
		return nil, scim.ErrInternalServer(fmt.Sprintf("failed to get user: %v", err))
	}

	return row.Data.User, nil
}

// ModifyUser updates a user's attributes
func (p *PostgresPlugin) ModifyUser(ctx context.Context, id string, patch *scim.PatchOp) error {
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
	query := `UPDATE users SET username = $1, data = $2, updated_at = $3 WHERE id = $4`

	userData := UserData{User: user}
	if _, err := p.db.ExecContext(ctx, query, user.UserName, userData, now, user.ID); err != nil {
		return scim.ErrInternalServer(fmt.Sprintf("failed to update user: %v", err))
	}

	return nil
}

// DeleteUser deletes a user
func (p *PostgresPlugin) DeleteUser(ctx context.Context, id string) error {
	result, err := p.db.ExecContext(ctx, "DELETE FROM users WHERE id = $1", id)
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

// GetGroups retrieves groups with optional filtering, sorting, and pagination
func (p *PostgresPlugin) GetGroups(ctx context.Context, params scim.QueryParams) ([]*scim.Group, error) {
	// Build optimized query from QueryParams
	qb := NewQueryBuilder("groups", "data", GroupAttributeMapping)
	query, args := qb.Build(params)

	// Rebind for PostgreSQL ($1, $2, etc.)
	query = p.db.Rebind(query)

	var rows []groupRow
	if err := p.db.SelectContext(ctx, &rows, query, args...); err != nil {
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
func (p *PostgresPlugin) CreateGroup(ctx context.Context, group *scim.Group) (*scim.Group, error) {
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
	err := p.db.GetContext(ctx, &exists, "SELECT EXISTS(SELECT 1 FROM groups WHERE display_name = $1)", group.DisplayName)
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
	query := `INSERT INTO groups (id, display_name, data, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`

	groupData := GroupData{Group: group}
	if _, err := p.db.ExecContext(ctx, query, group.ID, group.DisplayName, groupData, now, now); err != nil {
		return nil, scim.ErrInternalServer(fmt.Sprintf("failed to insert group: %v", err))
	}

	return group, nil
}

// GetGroup retrieves a specific group by ID
func (p *PostgresPlugin) GetGroup(ctx context.Context, id string, attributes []string) (*scim.Group, error) {
	var row groupRow
	query := `SELECT id, display_name, data, created_at, updated_at FROM groups WHERE id = $1`

	if err := p.db.GetContext(ctx, &row, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, scim.ErrNotFound("Group", id)
		}
		return nil, scim.ErrInternalServer(fmt.Sprintf("failed to get group: %v", err))
	}

	return row.Data.Group, nil
}

// ModifyGroup updates a group's attributes
func (p *PostgresPlugin) ModifyGroup(ctx context.Context, id string, patch *scim.PatchOp) error {
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
	query := `UPDATE groups SET display_name = $1, data = $2, updated_at = $3 WHERE id = $4`

	groupData := GroupData{Group: group}
	if _, err := p.db.ExecContext(ctx, query, group.DisplayName, groupData, now, group.ID); err != nil {
		return scim.ErrInternalServer(fmt.Sprintf("failed to update group: %v", err))
	}

	return nil
}

// DeleteGroup deletes a group
func (p *PostgresPlugin) DeleteGroup(ctx context.Context, id string) error {
	result, err := p.db.ExecContext(ctx, "DELETE FROM groups WHERE id = $1", id)
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

func (p *PostgresPlugin) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := p.db.PingContext(ctx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	return nil
}
