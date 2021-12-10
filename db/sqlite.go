package db

import (
	"database/sql"
	"log"
	"strconv"
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

	statement, _ := database.Prepare(user_table)
	statement.Exec()
	defer statement.Close()

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
		log.Println("Error retrieving last insert ID")
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
		log.Println("Error preparing icrement times first statement: ", err)
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
