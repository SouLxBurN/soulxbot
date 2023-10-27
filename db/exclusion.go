package db

import "log"

type Exclusion struct {
	ID         int
	StreamerID *int
	Username   string
}

func (d *Database) IsUserOnExclusionList(streamerID *int, username string) bool {
	statement, err := d.db.Prepare(CHECK_EXCLUSION_LIST)
	if statement != nil {
		defer func() { _ = statement.Close() }()
	}
	if err != nil {
		log.Println("Error preparing check exclusion list statement: ", err)
		return true
	}

	var count int
	err = statement.QueryRow(streamerID, username).Scan(&count)
	if err != nil {
		log.Println("Error checking if user should be excluded: ", err)
		return true
	}
	if count > 0 {
		return true
	}

	return false
}

// Insert username to exlude
func (d *Database) InsertExcludedUser(streamerID *int, username string) *Exclusion {
	statement, err := d.db.Prepare(INSERT_EXCLUDED_USER)
	if statement != nil {
		defer func() { _ = statement.Close() }()
	}
	if err != nil {
		log.Println("Error preparing insert excluded user statement: ", err)
		return nil
	}

	result, err := statement.Exec(streamerID, username)
	if err != nil {
		log.Println("Error inserting excluded user: ", err)
		return nil
	}

	newID, err := result.LastInsertId()

	return &Exclusion{
		ID:         int(newID),
		StreamerID: streamerID,
		Username:   username,
	}
}

const INSERT_EXCLUDED_USER = `
INSERT INTO exclusion(streamer_id, username)
VALUES
(?,?)`

const CHECK_EXCLUSION_LIST string = `
SELECT count(*)
FROM exclusion
WHERE (streamer_id=? OR streamer_id IS NULL) AND lower(username)=?
`

const FIND_STREAM_EXCLUSION_LIST string = `
SELECT username
FROM exclusion
WHERE streamer_id=? OR streamer_id=null
`

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
