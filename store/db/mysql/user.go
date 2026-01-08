package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/sagarsuperuser/userprofile/store"
)

func (d *DB) CreateLocalUser(ctx context.Context, in *store.CreateLocalUser) (*store.UserInfo, error) {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// users row (local => email is editable later, so email_locked=false)
	res, err := tx.ExecContext(ctx, `
		INSERT INTO users (email, email_locked, status, role)
		VALUES (?, FALSE, ?, ?)
	`, in.Email, in.Status, in.Role)
	if err != nil {
		return nil, err
	}

	userID, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	// local identity row
	_, err = tx.ExecContext(ctx, `
		INSERT INTO auth_identities (user_id, provider, password_hash)
		VALUES (?, 'local', ?)
	`, userID, in.PasswordHash)
	if err != nil {
		return nil, err
	}

	// profile row stub
	_, err = tx.ExecContext(ctx, `INSERT INTO user_profiles (user_id) VALUES (?)`, userID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return d.GetUser(ctx, &store.FindUser{ID: &userID})
}

func (d *DB) UpsertGoogleUser(ctx context.Context, email, sub string) (*store.UserInfo, error) {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// 1) If google sub already linked -> use same user
	var userID int64
	err = tx.QueryRowContext(ctx, `
		SELECT user_id
		FROM auth_identities
		WHERE provider = 'google' AND provider_subject = ?
		FOR UPDATE
	`, sub).Scan(&userID)

	if err == nil {
		// TODO - check if sync email is needed.
		// may cause issues if user already registered locally.

		return d.GetUser(ctx, &store.FindUser{ID: &userID})
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	// 2) sub not linked yet -> find user by email (lock if exists)
	err = tx.QueryRowContext(ctx, `
		SELECT id
		FROM users
		WHERE email = ?
		FOR UPDATE
	`, email).Scan(&userID)

	if errors.Is(err, sql.ErrNoRows) {
		// Create new google user
		res, err := tx.ExecContext(ctx, `
			INSERT INTO users (email, email_locked, status, role)
			VALUES (?, TRUE, ?, ?)
		`, email, store.StatusActive, store.RoleUser)
		if err != nil {
			return nil, err
		}
		userID, err = res.LastInsertId()
		if err != nil {
			return nil, err
		}

		if _, err := tx.ExecContext(ctx, `INSERT INTO user_profiles (user_id) VALUES (?)`, userID); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	// 3) Link google identity (idempotent)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO auth_identities (user_id, provider, provider_subject)
		VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE updated_at = CURRENT_TIMESTAMP
	`, userID, store.ProviderGoogle, sub)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return d.GetUser(ctx, &store.FindUser{ID: &userID})
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
			return nil, store.ErrEmailUpdateNotAllowed
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
		query := "UPDATE users SET " + strings.Join(set, ", ") + " WHERE id = ?"
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

	if v := find.Provider; v != nil {
		where, args = append(where, "ai.provider = ?"), append(args, *v)
	}

	joins := []string{
		"JOIN user_profiles p ON u.id = p.user_id",
		"JOIN auth_identities ai ON ai.user_id = u.id",
	}

	orderBy := []string{"u.created_at DESC"}

	query := `
		SELECT
		u.id,
		u.email,
		u.email_locked,
		u.status,
		u.role,
		ai.password_hash,
		p.full_name,
		p.telephone,
		p.avatar_url,
		u.created_at,
		u.updated_at
		FROM users u
		` + strings.Join(joins, "\n") + `
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

	var passwordHash sql.NullString
	list := make([]*store.UserInfo, 0)
	for rows.Next() {
		var user store.UserInfo
		if err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.EmailLocked,
			&user.Status,
			&user.Role,
			&passwordHash,
			&user.FullName,
			&user.Telephone,
			&user.AvatarURL,
			&user.CreatedAt,
			&user.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if passwordHash.Valid {
			user.PasswordHash = passwordHash.String
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
		return nil, store.ErrUserNotFound
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
