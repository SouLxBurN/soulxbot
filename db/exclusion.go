package db

type exclusion struct {
	ID         int
	StreamerID int
	Username   string
}

const exclusion_table = `
CREATE TABLE IF NOT EXISTS exclusion (
    id INTEGER PRIMARY KEY,
    streamer_id INTEGER,
    username TEXT,
    FOREIGN KEY (streamer_id)
    REFERENCES user (id)
        ON UPDATE CASCADE
        ON DELETE CASCADE
    )`
