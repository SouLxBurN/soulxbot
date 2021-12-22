package db

import "time"

const enable_foreign_keys string = `PRAGMA foreign_keys = ON`

const user_table string = `
CREATE TABLE IF NOT EXISTS user (
    id INTEGER PRIMARY KEY,
    username TEXT,
    timesFirst INTEGER DEFAULT 0
)`

const stream_table string = `
CREATE TABLE IF NOT EXISTS stream (
    id INTEGER PRIMARY KEY,
    title TEXT,
    startedAt DATETIME,
    first_userId INTEGER,
    FOREIGN KEY (first_userId)
    REFERENCES user (id)
        ON UPDATE SET NULL
        ON DELETE SET NULL
)`

type User struct {
	ID         int
	Username   string
	TimesFirst int
}

type Stream struct {
	ID        int
	Title     string
	StartedAt time.Time
	FirstUser *User
}
