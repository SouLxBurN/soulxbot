package db

import (
	"database/sql"
	"log"
)

const enable_foreign_keys string = `PRAGMA foreign_keys = ON`

type Database struct {
	db *sql.DB
}

// InitDatabase
func InitDatabase() *Database {
	database, err := sql.Open("sqlite3", "./irc.db")
	if err != nil {
		log.Println(err)
	}
	db := &Database{
		db: database,
	}

	if _, err := prepareAndExec(database, enable_foreign_keys); err != nil {
		log.Println("enable statement failed: ", err)
	}

	if _, err := prepareAndExec(database, user_table); err != nil {
		log.Println("create user_table failed: ", err)
	}

	if _, err := prepareAndExec(database, question_table); err != nil {
		log.Println("create question_table failed: ", err)
	}

	if _, err := prepareAndExec(database, stream_table); err != nil {
		log.Println("create stream_table failed: ", err)
	}

	if _, err := prepareAndExec(database, stream_config_table); err != nil {
		log.Println("create stream_config_table failed: ", err)
	}

	if _, err := prepareAndExec(database, exclusion_table); err != nil {
		log.Println("create exclusion_table failed: ", err)
	}

	migrateExistingStreamUsers(database)

	seedQuestionData(database)
	seedUserData(database)
	addQuestionDisabledColumn(database)
	seedExclusionList(database)
	addAuthToStreamConfig(database)
	migrateUserApiKeys(database)

	return db
}

func seedExclusionList(db *sql.DB) {
	check := `SELECT count(*) FROM exclusion`
	rows, err := db.Query(check)
	defer func() { _ = rows.Close() }()
	if err != nil {
		log.Println("failed empty exlusion check")
		return
	}

	rows.Next()
	var exclusion_count int
	rows.Scan(&exclusion_count)
	// Have to close the rows, otherwise database is locked.
	rows.Close()

	if exclusion_count <= 0 {
		if _, err := prepareAndExec(db, exclusionListSeed); err != nil {
			log.Println("exclusion list seed failed: ", err)
		}
	}
}

func migrateExistingStreamUsers(db *sql.DB) {
	check := `SELECT count(*) FROM stream_config`
	rows, err := db.Query(check)
	defer func() { _ = rows.Close() }()
	if err != nil {
		log.Println("failed empty question check")
		return
	}

	rows.Next()
	var config_count int
	rows.Scan(&config_count)
	// Have to close the rows, otherwise database is locked.
	rows.Close()

	if config_count <= 0 {
		if _, err := prepareAndExec(db, migrateConfigs); err != nil {
			log.Println("stream_config migrate failed: ", err)
		}
	}
}

func migrateUserApiKeys(db *sql.DB) {
	columnCheck := `SELECT count(*) as apiKeys FROM pragma_table_info('user') WHERE name = 'apiKey'`
	countCheck := `SELECT count(*) FROM user WHERE apiKey IS NOT NULL`
	moveApiKey := `UPDATE stream_config SET apiKey=u.apiKey FROM user u WHERE stream_config.userId=u.id AND u.apiKey IS NOT NULL`
	deleteOldKeys := `UPDATE user SET apiKey=NULL`

	rows, err := db.Query(columnCheck)
	defer func() { _ = rows.Close() }()
	if err != nil {
		log.Println("failed apiKey check")
		return
	}

	rows.Next()
	var column_count int
	rows.Scan(&column_count)
	// Have to close the rows, otherwise database is locked.
	rows.Close()

	rows, err = db.Query(countCheck)
	defer func() { _ = rows.Close() }()
	if err != nil {
		log.Println("failed count apiKey check")
		return
	}

	rows.Next()
	var config_count int
	rows.Scan(&config_count)
	// Have to close the rows, otherwise database is locked.
	rows.Close()

	if column_count > 0 && config_count > 0 {
		if _, err := prepareAndExec(db, moveApiKey); err != nil {
			log.Println("apiKey migrate failed: ", err)
		}
		if _, err := prepareAndExec(db, deleteOldKeys); err != nil {
			log.Println("delete user.apiKey script failed: ", err)
		}
	}
}

func seedQuestionData(db *sql.DB) {
	questionCheck := `SELECT count(*) FROM question`
	rows, err := db.Query(questionCheck)
	defer func() { _ = rows.Close() }()
	if err != nil {
		log.Println("failed empty question check")
		return
	}

	rows.Next()
	var questionCount int
	rows.Scan(&questionCount)
	// Have to close the rows, otherwise database is locked.
	rows.Close()
	if questionCount <= 0 {
		if _, err := prepareAndExec(db, questionSeed); err != nil {
			log.Println("questionseed failed: ", err)
		}
	}
}

func seedUserData(db *sql.DB) {
	userCheck := `SELECT count(*) FROM user`
	rows, err := db.Query(userCheck)
	defer func() { _ = rows.Close() }()
	if err != nil {
		log.Println("failed empty user check")
		return
	}

	rows.Next()
	var userCount int
	rows.Scan(&userCount)
	// Have to close the rows, otherwise database is locked.
	rows.Close()
	if userCount <= 0 {
		if _, err := prepareAndExec(db, userSeed); err != nil {
			log.Println("userSeed failed: ", err)
		}
		if _, err := prepareAndExec(db, streamConfigSeed); err != nil {
			log.Println("streamConfigSeed failed: ", err)
		}
	}
}

// Migration Script for adding disabled column to question table
func addQuestionDisabledColumn(db *sql.DB) {
	disableCheck := `SELECT count(*) as disabled FROM pragma_table_info('question') WHERE name = 'disabled';`
	addDisabledColumn := `ALTER TABLE question ADD COLUMN disabled int default false`
	addSkipCountColumn := `ALTER TABLE question ADD COLUMN skipCount int default 0`
	rows, err := db.Query(disableCheck)
	defer func() { _ = rows.Close() }()
	if err != nil {
		log.Println("question.disabled column check failed")
		return
	}

	rows.Next()
	var disabledPresent int
	rows.Scan(&disabledPresent)
	// Have to close the rows, otherwise database is locked.
	rows.Close()
	if disabledPresent == 0 {
		if _, err := prepareAndExec(db, addDisabledColumn); err != nil {
			log.Println("question.disabled column script failed: ", err)
		}
		if _, err := prepareAndExec(db, addSkipCountColumn); err != nil {
			log.Println("question.disabled column script failed: ", err)
		}
	}
}

// Migration Script for adding disabled column to question table
func addAuthToStreamConfig(db *sql.DB) {
	authCheck := `SELECT count(*) as disabled FROM pragma_table_info('stream_config') WHERE name = 'twitchAuthToken';`
	addApiKeyColumn := `ALTER TABLE stream_config ADD COLUMN apiKey TEXT`
	addAuthTokenColumn := `ALTER TABLE stream_config ADD COLUMN twitchAuthToken TEXT`
	addRefreshTokenColumn := `ALTER TABLE stream_config ADD COLUMN twitchRefreshToken TEXT`

	rows, err := db.Query(authCheck)
	defer func() { _ = rows.Close() }()
	if err != nil {
		log.Println("stream_config.twitchAuthToken column check failed")
		return
	}

	rows.Next()
	var authTokenPresent int
	rows.Scan(&authTokenPresent)
	// Have to close the rows, otherwise database is locked.
	rows.Close()

	if authTokenPresent == 0 {
		if _, err := prepareAndExec(db, addApiKeyColumn); err != nil {
			log.Println("stream_config.apiKey column script failed: ", err)
		}
		if _, err := prepareAndExec(db, addAuthTokenColumn); err != nil {
			log.Println("stream_config.twitchAuthToken column script failed: ", err)
		}
		if _, err := prepareAndExec(db, addRefreshTokenColumn); err != nil {
			log.Println("stream_config.twitchRefreshToken column script failed: ", err)
		}
	}
}

// Helper function to prepare, exec and close a query
func prepareAndExec(db *sql.DB, query string) (sql.Result, error) {
	statement, err := db.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer statement.Close()

	result, err := statement.Exec()
	if err != nil {
		return nil, err
	}

	return result, nil
}
