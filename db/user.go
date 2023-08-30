package db

import (
	"log"
	"time"
)

type User struct {
	ID          int
	Username    string
	DisplayName string
	APIKey      *string
}

type StreamConfig struct {
	ID           int
	UserId       int
	BotDisabled  bool
	FirstEnabled bool
	FirstEpoch   time.Time
	QotdEnabled  bool
	QotdEpoch    time.Time
	DateUpdated  time.Time
}

type StreamUser struct {
	User
	StreamConfig
}

// InsertUser
func (d *Database) InsertUser(id int, username string, displayName string) *User {
	statement, err := d.db.Prepare(INSERT_USER)
	if statement != nil {
		defer func() { _ = statement.Close() }()
	}
	if err != nil {
		log.Println("Error preparing insert user statement: ", err)
		return nil
	}

	_, err = statement.Exec(id, username, displayName)
	if err != nil {
		log.Println("Error inserting user: ", err)
		return nil
	}

	return &User{
		ID:          id,
		Username:    username,
		DisplayName: displayName,
	}
}

// UpdateUserName
func (d *Database) UpdateUserName(ID int, newUserName string, newDisplayName string) error {
	statement, err := d.db.Prepare(UPDATE_USERNAME)
	if statement != nil {
		defer func() { _ = statement.Close() }()
	}
	if err != nil {
		return err
	}

	_, err = statement.Exec(newUserName, newDisplayName, ID)
	if err != nil {
		return err
	}
	return nil
}

// FindAllUsers
func (d *Database) FindAllUsers() ([]User, error) {
	rows, err := d.db.Query(FIND_ALL_USERS)
	defer func() { _ = rows.Close() }()
	if err != nil {
		log.Println("Error finding all users: ", err)
		return nil, err
	}

	var users []User
	for rows.Next() {
		var user User
		rows.Scan(&user.ID, &user.Username, &user.DisplayName)
		users = append(users, user)
	}

	return users, nil
}

// FindAllApiKeyUsers
func (d *Database) FindAllApiKeyUsers() ([]User, error) {
	rows, err := d.db.Query(FIND_ALL_APIKEY_USERS)
	defer func() { _ = rows.Close() }()

	if err != nil {
		log.Println("Error finding registered users : ", err)
		return nil, err
	}

	var results []User
	for rows.Next() {
		var user User
		rows.Scan(&user.ID, &user.Username, &user.DisplayName)
		results = append(results, user)
	}

	return results, nil
}

// UpdateAPIKeyForUser
func (d *Database) UpdateAPIKeyForUser(userId int, apiKey string) error {
	statement, err := d.db.Prepare(UPDATE_APIKEY_BY_USERID)
	if statement != nil {
		defer func() { _ = statement.Close() }()
	}
	if err != nil {
		log.Println("Error preparing update api key statement: ", err)
	}

	_, err = statement.Exec(apiKey, userId)
	if err != nil {
		log.Println("Error updating api key: ", err)
	}

	return nil
}

// FindUserByID
func (d *Database) FindUserByID(ID int) (*User, bool) {
	rows, _ := d.db.Query(FIND_USER_BY_ID, ID)
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return nil, false
	}

	var user User
	rows.Scan(&user.ID, &user.Username, &user.DisplayName)

	return &user, true
}

// FindUserByUsername
func (d *Database) FindUserByUsername(username string) (*User, bool) {
	rows, err := d.db.Query(FIND_USER_BY_USERNAME, username)
	defer func() { _ = rows.Close() }()
	if err != nil {
		log.Println("Error finding user: ", err)
		return nil, false
	}
	if !rows.Next() {
		return nil, false
	}

	var user User
	rows.Scan(&user.ID, &user.Username, &user.DisplayName)

	return &user, true
}

// FindUserByApiKey
func (d *Database) FindUserByApiKey(apiKey string) (*User, bool) {
	rows, err := d.db.Query(FIND_USER_BY_APIKEY, apiKey)
	defer func() { _ = rows.Close() }()
	if err != nil {
		log.Println("Error finding user: ", err)
		return nil, false
	}
	if !rows.Next() {
		return nil, false
	}

	var user User
	rows.Scan(&user.ID, &user.Username, &user.DisplayName)

	return &user, true
}

// CreateStreamConfig
func (d *Database) CreateStreamConfig(userId int) (*StreamConfig, error) {
	statement, err := d.db.Prepare(CREATE_STREAM_CONFIG)
	if statement != nil {
		defer func() { _ = statement.Close() }()
	}
	if err != nil {
		log.Println("Error preparing insert stream config statement: ", err)
		return nil, err
	}

	config := StreamConfig{
		UserId:       userId,
		BotDisabled:  false,
		FirstEnabled: true,
		FirstEpoch:   time.Now(),
		QotdEnabled:  true,
		QotdEpoch:    time.Now(),
		DateUpdated:  time.Now(),
	}

	result, err := statement.Exec(
		userId,
		config.BotDisabled,
		config.FirstEnabled,
		config.FirstEpoch,
		config.QotdEnabled,
		config.QotdEpoch,
		config.DateUpdated,
	)
	if err != nil {
		log.Println("Error creating stream config", err)
		return nil, err
	}

	newID, err := result.LastInsertId()
	config.ID = int(newID)

	return &config, nil
}

// FindStreamUserByUserID
func (d *Database) FindStreamUserByUserID(userId int) (*StreamUser, error) {
	rows, err := d.db.Query(FIND_STREAM_USER_BY_USERID, userId)
	defer func() { _ = rows.Close() }()
	if err != nil {
		log.Println("stream user by id query failed")
		return nil, err
	}

	if !rows.Next() {
		return nil, nil
	}

	var user User
	var config StreamConfig
	rows.Scan(
		&user.ID,
		&user.Username,
		&user.DisplayName,
		&config.ID,
		&config.UserId,
		&config.BotDisabled,
		&config.FirstEnabled,
		&config.FirstEpoch,
		&config.QotdEnabled,
		&config.QotdEpoch,
		&config.DateUpdated)

	return &StreamUser{
		user,
		config,
	}, nil
}

// FindStreamUserByUsername
func (d *Database) FindStreamUserByUserName(username string) (*StreamUser, error) {
	rows, err := d.db.Query(FIND_STREAM_USER_BY_USERNAME, username)
	if err != nil {
		log.Println("stream user by username query failed")
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return nil, nil
	}

	var user User
	var config StreamConfig
	rows.Scan(
		&user.ID,
		&user.Username,
		&user.DisplayName,
		&config.ID,
		&config.UserId,
		&config.BotDisabled,
		&config.FirstEnabled,
		&config.FirstEpoch,
		&config.QotdEnabled,
		&config.QotdEpoch,
		&config.DateUpdated)

	return &StreamUser{
		user,
		config,
	}, nil
}

const user_table string = `
CREATE TABLE IF NOT EXISTS user (
    id INTEGER PRIMARY KEY,
    username TEXT,
    displayName TEXT,
    apiKey TEXT
    )`

const stream_config_table = `
CREATE TABLE IF NOT EXISTS stream_config (
    id INTEGER PRIMARY KEY,
    userId INTEGER UNIQUE,
    botDisabled bool,
    firstEnabled bool,
    firstEpoch DATETIME,
    qotdEnabled bool,
    qotdEpoch DATETIME,
    dateUpdated DATATIME,
    FOREIGN KEY (userId)
    REFERENCES user (id)
        ON UPDATE CASCADE
        ON DELETE CASCADE
    )`
