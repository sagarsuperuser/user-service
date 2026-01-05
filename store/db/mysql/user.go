package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/sagarsuperuser/userprofile/store"
)

func (d *DB) CreateUser(ctx context.Context, create *store.CreateUser) (*store.UserInfo, error) {
	// validate provider shape
	if err := validateProviderShape(create); err != nil {
		return nil, err
	}

	// start transaction
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// insert into user table
	fields := []string{"email", "email_locked"}
	args := []any{create.Email, create.EmailLocked}
	if create.Status != nil {
		fields = append(fields, "status")
		args = append(args, *create.Status)
	}
	if create.Role != nil {
		fields = append(fields, "role")
		args = append(args, *create.Role)
	}
	stmt := "INSERT INTO users (" + strings.Join(fields, ", ") + ") VALUES (" + placeholders(len(args)) + ")"
	result, err := tx.ExecContext(ctx, stmt, args...)
	if err != nil {
		return nil, err
	}

	// insert into auth_identities table
	userId, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	fields = []string{"user_id", "provider"}
	args = []any{userId, create.Provider}
	if create.ProviderSubject != nil {
		fields = append(fields, "provider_subject")
		args = append(args, *create.ProviderSubject)
	}
	if create.PasswordHash != nil {
		fields = append(fields, "password_hash")
		args = append(args, create.PasswordHash)
	}
	stmt = "INSERT INTO auth_identities (" + strings.Join(fields, ", ") + ") VALUES (" + placeholders(len(args)) + ")"
	_, err = tx.ExecContext(ctx, stmt, args...)
	if err != nil {
		return nil, err
	}

	// insert into user_profiles table
	_, err = tx.ExecContext(ctx, "INSERT INTO user_profiles (user_id) VALUES (?)", userId)
	if err != nil {
		return nil, err
	}

	// commit transaction
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// fetch and return the created user
	return d.GetUser(ctx, &store.FindUser{ID: &userId})

}

func (d *DB) UpdateUser(ctx context.Context, update *store.UpdateUser) (*store.UserInfo, error) {
	// check if email updation is locked
	if update.Email != nil {
		var emailLocked bool
		err := d.db.QueryRowContext(ctx, "SELECT email_locked FROM users WHERE id = ?", update.ID).Scan(&emailLocked)
		if err != nil {
			return nil, err
		}
		if emailLocked {
			return nil, fmt.Errorf("update email is locked")
		}
	}

	// start transaction
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// update user_profiles table
	set := []string{}
	args := []any{}
	if v := update.FullName; v != nil {
		set, args = append(set, "full_name = ?"), append(args, *v)
	}
	if v := update.Telephone; v != nil {
		set, args = append(set, "telephone = ?"), append(args, *v)
	}
	if v := update.AvatarURL; v != nil {
		set, args = append(set, "avatar_url = ?"), append(args, *v)
	}

	if len(set) > 0 {
		args = append(args, update.ID)
		query := "UPDATE user_profiles SET " + strings.Join(set, ", ") + " WHERE user_id = ?"
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return nil, err
		}
	}

	// update user table
	set = []string{}
	args = []any{}
	if v := update.Role; v != nil {
		set, args = append(set, "role = ?"), append(args, *v)
	}

	if v := update.Email; v != nil {
		set, args = append(set, "email = ?"), append(args, *v)
	}

	if len(set) > 0 {
		args = append(args, update.ID)
		query := "UPDATE user SET " + strings.Join(set, ", ") + " WHERE id = ?"
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return nil, err
		}
	}

	// commit transaction
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// fetch and return the updated user
	return d.GetUser(ctx, &store.FindUser{ID: &update.ID})
}

func (d *DB) ListUsers(ctx context.Context, find *store.FindUser) ([]*store.UserInfo, error) {
	where, args := []string{"1 = 1"}, []any{}

	// user table where clauses
	if v := find.ID; v != nil {
		where, args = append(where, "u.id = ?"), append(args, *v)
	}

	if v := find.Email; v != nil {
		where, args = append(where, "u.email = ?"), append(args, *v)
	}

	if v := find.Role; v != nil {
		where, args = append(where, "u.role = ?"), append(args, *v)
	}

	orderBy := []string{"u.created_at DESC"}

	query := `
		SELECT
		u.id,
		u.email,
		u.email_locked,
		u.status,
		u.role,
		p.full_name,
		p.telephone,
		p.avatar_url,
		u.created_at,
		u.updated_at,
		FROM users u
		JOIN user_profiles p ON u.id = p.user_id
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY ` + strings.Join(orderBy, ", ")
	if v := find.Limit; v != nil {
		query += fmt.Sprintf(" LIMIT %d", *v)
	}
	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*store.UserInfo, 0)
	for rows.Next() {
		var user store.UserInfo
		if err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.EmailLocked,
			&user.Status,
			&user.Role,
			&user.FullName,
			&user.Telephone,
			&user.AvatarURL,
			&user.CreatedAt,
			&user.UpdatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, &user)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return list, nil
}

func (d *DB) GetUser(ctx context.Context, find *store.FindUser) (*store.UserInfo, error) {
	list, err := d.ListUsers(ctx, find)
	if err != nil {
		return nil, err
	}
	if len(list) != 1 {
		return nil, fmt.Errorf("unexpected user count: %d", len(list))
	}
	return list[0], nil
}

func (d *DB) DeleteUser(ctx context.Context, delete *store.DeleteUser) (bool, error) {
	result, err := d.db.ExecContext(ctx, "DELETE FROM user WHERE id = ?", delete.ID)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return n == 1, nil
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	list := make([]string, n)
	for i := range n {
		list[i] = "?"
	}

	return strings.Join(list, ", ")

}

func validateProviderShape(u *store.CreateUser) error {
	switch u.Provider {
	case store.ProviderLocal:
		if u.PasswordHash == nil || *u.PasswordHash == "" {
			return fmt.Errorf("password_hash required for local provider")
		}
		if u.ProviderSubject != nil {
			return fmt.Errorf("provider_subject must be empty for local provider")
		}
	case store.ProviderGoogle:
		if u.ProviderSubject == nil || *u.ProviderSubject == "" {
			return fmt.Errorf("provider_subject required for google provider")
		}
		if u.PasswordHash != nil {
			return fmt.Errorf("password_hash must be empty for google provider")
		}
	default:
		return fmt.Errorf("invalid provider")
	}
	return nil
}
