package db

const user_table string = `
CREATE TABLE IF NOT EXISTS user (
    id INTEGER PRIMARY KEY,
    username TEXT,
    timesFirst INTEGER DEFAULT 0
)`

type User struct {
	ID         int
	Username   string
	TimesFirst int
}
