package auth

import (
	"database/sql"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type Role string

const (
	RoleAdmin  Role = "admin"
	RoleViewer Role = "viewer"
)

type User struct {
	ID        int64
	Username  string
	Role      Role
	CreatedAt time.Time
}

type UserStore struct {
	db *sql.DB
}

func NewUserStore(db *sql.DB) *UserStore {
	return &UserStore{db: db}
}

func (s *UserStore) Init() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id         BIGINT PRIMARY KEY,
			username   VARCHAR UNIQUE NOT NULL,
			password   VARCHAR NOT NULL,
			role       VARCHAR NOT NULL DEFAULT 'viewer',
			created_at TIMESTAMP DEFAULT NOW()
		);
		CREATE SEQUENCE IF NOT EXISTS users_seq START 1;
	`)
	return err
}

func (s *UserStore) Create(username, password string, role Role) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, err
	}
	var id int64
	err = s.db.QueryRow(`
		INSERT INTO users (id, username, password, role)
		VALUES (nextval('users_seq'), ?, ?, ?)
		RETURNING id
	`, username, string(hash), role).Scan(&id)
	if err != nil {
		return nil, err
	}
	return &User{ID: id, Username: username, Role: role}, nil
}

func (s *UserStore) Verify(username, password string) (*User, error) {
	var u User
	var hash string
	err := s.db.QueryRow(`
		SELECT id, username, password, role FROM users WHERE username = ?
	`, username).Scan(&u.ID, &u.Username, &hash, &u.Role)
	if err == sql.ErrNoRows {
		return nil, errors.New("invalid credentials")
	}
	if err != nil {
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}
	return &u, nil
}

func (s *UserStore) GetByID(id int64) (*User, error) {
	var u User
	err := s.db.QueryRow(`
		SELECT id, username, role FROM users WHERE id = ?
	`, id).Scan(&u.ID, &u.Username, &u.Role)
	if err == sql.ErrNoRows {
		return nil, errors.New("user not found")
	}
	return &u, err
}

func (s *UserStore) List() ([]User, error) {
	rows, err := s.db.Query(`SELECT id, username, role, created_at FROM users ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.CreatedAt); err != nil {
			continue
		}
		users = append(users, u)
	}
	return users, nil
}

func (s *UserStore) Count() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}
