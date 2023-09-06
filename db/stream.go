package db

import (
	"database/sql"
	"log"
	"time"
)

type Stream struct {
	ID          int
	TWID        *int
	UserId      int
	Title       *string
	StartedAt   time.Time
	EndedAt     *time.Time
	QOTDId      *int
	FirstUserId *int
}

type FirstLeadersResult struct {
	User       User
	TimesFirst int
}

// FindCurrentStream
func (d *Database) FindCurrentStream(userId int) *Stream {
	rows, err := d.db.Query(FIND_CURRENT_STREAM_BY_USERID, userId)
	defer func() { _ = rows.Close() }()
	if err != nil {
		log.Println("Error finding current stream statement: ", err)
		return nil
	}
	var stream Stream
	if rows.Next() {
		scanRowIntoStream(&stream, rows)
		return &stream
	} else {
		return nil
	}
}

// FindAllCurrentStreams
func (d *Database) FindAllCurrentStreams() []Stream {
	rows, err := d.db.Query(FIND_ALL_CURRENT_STREAMS)
	defer func() { _ = rows.Close() }()
	if err != nil {
		log.Println("Error finding all current streams statement: ", err)
		return nil
	}
	var streams []Stream

	for rows.Next() {
		var stream Stream
		scanRowIntoStream(&stream, rows)
		streams = append(streams, stream)
	}
	return streams
}

// Helper for scanning rows into a stream
func scanRowIntoStream(stream *Stream, rows *sql.Rows) {
	rows.Scan(
		&stream.ID,
		&stream.TWID,
		&stream.Title,
		&stream.StartedAt,
		&stream.EndedAt,
		&stream.UserId,
		&stream.FirstUserId,
		&stream.QOTDId,
	)
}

func (d *Database) FindStreamById(streamId int) *Stream {
	rows, err := d.db.Query(FIND_STREAM_BY_ID, streamId)
	defer func() { _ = rows.Close() }()
	if err != nil {
		log.Println("Error finding stream by id statement: ", err)
		return nil
	}
	var stream Stream
	if rows.Next() {
		scanRowIntoStream(&stream, rows)
		return &stream
	} else {
		return nil
	}
}

// InsertStream
// Inserts a new stream record with with most data as null
func (d *Database) InsertStream(userId int, startedAt time.Time) *Stream {
	statement, err := d.db.Prepare(INSERT_STREAM)
	if statement != nil {
		defer func() { _ = statement.Close() }()
	}
	if err != nil {
		log.Println("Error preparing insert stream statement: ", err)
		return nil
	}

	result, err := statement.Exec(userId, startedAt)
	if err != nil {
		log.Println("Error inserting stream: ", err)
		return nil
	}

	newID, err := result.LastInsertId()
	if err != nil {
		log.Println("Error retrieving last insert stream ID")
		return nil
	}

	return &Stream{
		ID:        int(newID),
		UserId:    userId,
		StartedAt: startedAt,
	}
}

// UpdateFirstUser
func (d *Database) UpdateFirstUser(streamId int, userId int) error {
	statement, err := d.db.Prepare(UPDATE_FIRST_USER)
	if statement != nil {
		defer func() { _ = statement.Close() }()
	}
	if err != nil {
		log.Println("Error preparing update first user statement: ", err)
		return err
	}

	result, err := statement.Exec(userId, streamId)
	if err != nil {
		log.Printf("Error updating first user for streamId(%d): %x\n", streamId, err)
		return err
	}

	_, err = result.RowsAffected()
	if err != nil {
		log.Println("Error retrieving rows affected")
		return nil
	}

	return nil
}

func (d *Database) UpdateStreamEndedAt(streamId int, endedAt time.Time) error {
	statement, err := d.db.Prepare(UPDATE_STREAM_ENDED)
	if statement != nil {
		defer func() { _ = statement.Close() }()
	}
	if err != nil {
		log.Println("Error preparing update stream endedAt statement: ", err)
		return err
	}

	_, err = statement.Exec(endedAt, streamId)
	if err != nil {
		log.Printf("Error updating stream endedAt for streamId(%d): %x\n", streamId, err)
		return err
	}

	return nil
}

func (d *Database) UpdateStreamInfo(streamId int, twid int, title string) error {
	statement, err := d.db.Prepare(UPDATE_STREAM_INFO)
	if statement != nil {
		defer func() { _ = statement.Close() }()
	}
	if err != nil {
		log.Println("Error preparing update stream info statement: ", err)
		return err
	}

	_, err = statement.Exec(twid, title, streamId)
	if err != nil {
		log.Printf("Error updating stream info for streamId(%d): %x\n", streamId, err)
		return err
	}

	return nil
}

// UpdateStreamQuestion
func (d *Database) UpdateStreamQuestion(streamId int, questionId *int) error {
	statement, err := d.db.Prepare(UPDATE_STREAM_QUESTION)
	if statement != nil {
		defer func() { _ = statement.Close() }()
	}
	if err != nil {
		log.Println("Error preparing update stream question statement: ", err)
		return err
	}

	_, err = statement.Exec(questionId, streamId)
	if err != nil {
		log.Printf("Error updating stream question for streamId(%d): %x\n", streamId, err)
		return err
	}

	return nil
}

// FindFirstLeaders
func (d *Database) FindFirstLeaders(streamUser int, count int) ([]FirstLeadersResult, error) {
	rows, err := d.db.Query(FIND_TIMES_FIRST_LEADERS, streamUser, count)
	defer func() { _ = rows.Close() }()
	if err != nil {
		log.Println("Error finding first leaders: ", err)
		return nil, err
	}

	var results []FirstLeadersResult
	for rows.Next() {
		var user User
		var timesFirst int
		rows.Scan(&user.ID, &user.Username, &user.DisplayName, &timesFirst)
		results = append(results, FirstLeadersResult{user, timesFirst})
	}

	return results, nil
}

// FindUserTimesFirst
func (d *Database) FindUserTimesFirst(streamUserId int, userId int) (int, error) {
	rows, err := d.db.Query(FIND_USER_TIMES_FIRST, streamUserId, userId)
	defer func() { _ = rows.Close() }()
	if err != nil {
		log.Println("Error finding first count: ", err)
		return 0, err
	}

	var timesFirst int
	if rows.Next() {
		rows.Scan(&timesFirst)
	}

	return timesFirst, nil
}

const stream_table string = `
CREATE TABLE IF NOT EXISTS stream (
    id INTEGER PRIMARY KEY,
    twid INTEGER,
    title TEXT,
    startedAt DATETIME,
    endedAt DATETIME,
    userId INTEGER NOT NULL,
    first_userId INTEGER,
    qotdId INTEGER,
    FOREIGN KEY (userId)
    REFERENCES user (id)
        ON UPDATE CASCADE
        ON DELETE CASCADE
    FOREIGN KEY (first_userId)
    REFERENCES user (id)
        ON UPDATE SET NULL
        ON DELETE SET NULL
    FOREIGN KEY (qotdId)
    REFERENCES question (id)
        ON UPDATE SET NULL
        ON DELETE SET NULL
    )`
