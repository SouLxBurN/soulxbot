package db

import (
	"database/sql"
	"log"
	"strconv"
	"time"
)

type Database struct {
	db *sql.DB
}

// InitDatabase
func InitDatabase() *Database {
	database, err := sql.Open("sqlite3", "./irc.db")
	if err != nil {
		log.Println(err)
	}

	statement, err := database.Prepare(enable_foreign_keys)
	if err != nil {
		log.Println("Enable statement failed: ", err)
	}
	statement.Exec()
	statement.Close()

	statement, _ = database.Prepare(user_table)
	statement.Exec()
	statement.Close()

	statement, err = database.Prepare(stream_table)
	if err != nil {
		log.Println("Prepared statement failed: ", err)
	}
	statement.Exec()
	statement.Close()

	return &Database{
		db: database,
	}
}

// InsertUser
func (d *Database) InsertUser(username string) *User {
	statement, err := d.db.Prepare(INSERT_USER)
	defer statement.Close()
	if err != nil {
		log.Println("Error preparing insert user statement: ", err)
		return nil
	}

	result, err := statement.Exec(username)
	if err != nil {
		log.Println("Error inserting user: ", err)
		return nil
	}

	newID, err := result.LastInsertId()
	if err != nil {
		log.Println("Error retrieving last insert user ID")
		return nil
	}

	return &User{
		ID:         int(newID),
		Username:   username,
		TimesFirst: 0,
	}

}

// IncrementTimesFirst
func (d *Database) IncrementTimesFirst(ID int) error {
	statement, err := d.db.Prepare(INCREMENT_TIMES_FIRST)
	defer statement.Close()
	if err != nil {
		log.Println("Error preparing increment times first statement: ", err)
		return err
	}

	result, err := statement.Exec(ID)
	if err != nil {
		log.Println("Error incrementing timesFirst: ", err)
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		log.Println("Error retrieving rows affected")
		return nil
	}

	log.Println(ID)
	log.Println(rows)

	return nil
}

// IncrementTimesFirst
func (d *Database) DecrementTimesFirst(ID int) error {
	statement, err := d.db.Prepare(DECREMENT_TIMES_FIRST)
	defer statement.Close()
	if err != nil {
		log.Println("Error preparing decrement times first statement: ", err)
		return err
	}

	result, err := statement.Exec(ID)
	if err != nil {
		log.Println("Error decrementing timesFirst: ", err)
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		log.Println("Error retrieving rows affected")
		return nil
	}

	log.Println(ID)
	log.Println(rows)

	return nil
}

// FindUserByID
func (d *Database) FindUserByID(ID int) (*User, bool) {
	rows, _ := d.db.Query(FIND_USER_BY_ID)
	defer rows.Close()
	if !rows.Next() {
		return nil, false
	}

	var user User
	rows.Scan(&user.ID, &user.Username, &user.TimesFirst)

	return &user, true
}

// FindUserByUsername
func (d *Database) FindUserByUsername(username string) (*User, bool) {
	rows, err := d.db.Query(FIND_USER_BY_USERNAME, username)
	defer rows.Close()
	if err != nil {
		log.Println("Error finding user: ", err)
		return nil, false
	}
	if !rows.Next() {
		return nil, false
	}

	var user User
	rows.Scan(&user.ID, &user.Username, &user.TimesFirst)

	return &user, true
}

// TimesFirstLeaders
func (d *Database) TimesFirstLeaders(num int) ([]User, error) {
	rows, err := d.db.Query(FIND_TIMES_FIRST_LEADERS, num)
	defer rows.Close()
	if err != nil {
		log.Println("Error finding times first leaders: ", err)
		return nil, err
	}

	var users []User
	for rows.Next() {
		var user User
		rows.Scan(&user.ID, &user.Username, &user.TimesFirst)
		users = append(users, user)
	}

	return users, nil
}

// FindAllUsers
func (d *Database) FindAllUsers() {
	rows, err := d.db.Query(FIND_ALL_USERS)
	defer rows.Close()
	if err != nil {
		log.Println("Error finding all users: ", err)
		return
	}

	var user User
	for rows.Next() {
		rows.Scan(&user.ID, &user.Username, &user.TimesFirst)
		log.Println(strconv.Itoa(user.ID) + " " + user.Username + " " + strconv.Itoa(user.TimesFirst))
	}
}

// InsertStream
// Inserts a new stream record with the first user as null
func (d *Database) InsertStream(title string, startedAt time.Time) *Stream {
	statement, err := d.db.Prepare(INSERT_STREAM)
	defer statement.Close()
	if err != nil {
		log.Println("Error preparing insert stream statement: ", err)
		return nil
	}

	result, err := statement.Exec(title, startedAt, nil)
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
		Title:     title,
		StartedAt: startedAt,
	}
}

// UpdateFirstUser
func (d *Database) UpdateFirstUser(streamId int, userId int) error {
	statement, err := d.db.Prepare(UPDATE_FIRST_USER)
	defer statement.Close()
	if err != nil {
		log.Println("Error preparing update first user statement: ", err)
		return err
	}

	result, err := statement.Exec(userId, streamId)
	if err != nil {
		log.Printf("Error updating first user for streamId(%d): %x\n", streamId, err)
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		log.Println("Error retrieving rows affected")
		return nil
	}

	log.Println(streamId, userId)
	log.Println(rows)

	return nil
}
