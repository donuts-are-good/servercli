package main

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	db *sqlx.DB
}

func NewDatabase(dbPath string) (*Database, error) {
	db, err := sqlx.Connect("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Create the users table if it doesn't exist.
	usersSchema := `
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY,
			username TEXT UNIQUE NOT NULL,
			password TEXT NOT NULL
		);
	`

	if _, err := db.Exec(usersSchema); err != nil {
		return nil, err
	}

	// Create the friends table if it doesn't exist.
	friendsSchema := `
		CREATE TABLE IF NOT EXISTS friends (
			user_id INTEGER NOT NULL,
			friend_id INTEGER NOT NULL,
			PRIMARY KEY (user_id, friend_id),
			FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
			FOREIGN KEY (friend_id) REFERENCES users (id) ON DELETE CASCADE
		);
	`

	if _, err := db.Exec(friendsSchema); err != nil {
		return nil, err
	}

	return &Database{db: db}, nil
}
func (d *Database) GetUser(username string) (*User, error) {
	user := &User{}
	err := d.db.Get(user, "SELECT * FROM users WHERE username = ?", username)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (d *Database) CreateUser(username, password string) error {
	_, err := d.db.Exec("INSERT INTO users (username, password) VALUES (?, ?)", username, password)
	return err
}

func (d *Database) GetFriendsList(username string) ([]string, error) {
	var friends []string
	err := d.db.Select(&friends, `
		SELECT u.username
		FROM users u
		JOIN friends f ON u.id = f.friend_id
		WHERE f.user_id = (SELECT id FROM users WHERE username = ?)
	`, username)
	if err != nil {
		return nil, err
	}
	return friends, nil
}

func (d *Database) AddFriend(username, friendUsername string) error {
	_, err := d.db.Exec(`
		INSERT INTO friends (user_id, friend_id)
		VALUES (
			(SELECT id FROM users WHERE username = ?),
			(SELECT id FROM users WHERE username = ?)
		)
	`, username, friendUsername)
	return err
}

func (d *Database) RemoveFriend(username, friendUsername string) error {
	_, err := d.db.Exec(`
		DELETE FROM friends
		WHERE
			user_id = (SELECT id FROM users WHERE username = ?) AND
			friend_id = (SELECT id FROM users WHERE username = ?)
	`, username, friendUsername)
	return err
}
