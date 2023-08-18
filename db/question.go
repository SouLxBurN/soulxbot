package db

import (
	"errors"
	"log"
)

type Question struct {
	ID        int    `json:"id"`
	Text      string `json:"text"`
	Disabled  bool   `json:"disabled"`
	SkipCount int    `json:"skipCount"`
}

// IncrementQuestionSkip
func (d *Database) IncrementQuestionSkip(questionId int) (int, error) {
	statement, err := d.db.Prepare(INCREMENT_QUESTION_SKIP)
	if statement != nil {
		defer func() { _ = statement.Close() }()
	}
	if err != nil {
		log.Println("Error preparing increment question skip statement: ", err)
		return 0, err
	}

	_, err = statement.Exec(questionId)
	if err != nil {
		log.Printf("Error incrementing question: %x\n", err)
		return 0, err
	}
	question, _ := d.FindQuestionByID(questionId)

	return question.SkipCount, nil
}

// DisableQuestion
func (d *Database) DisableQuestion(questionId int) error {
	statement, err := d.db.Prepare(DISABLE_QUESTION)
	if statement != nil {
		defer func() { _ = statement.Close() }()
	}
	if err != nil {
		log.Println("Error preparing disable question statement: ", err)
		return err
	}

	_, err = statement.Exec(questionId)
	if err != nil {
		log.Printf("Error marking question as disabled: %x\n", err)
		return err
	}

	return nil
}

// FindQuestionByID
func (d *Database) FindQuestionByID(ID int) (*Question, bool) {
	rows, _ := d.db.Query(FIND_QUESTION_BY_ID, ID)
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return nil, false
	}

	var question Question
	rows.Scan(&question.ID, &question.Text, &question.Disabled, &question.SkipCount)

	return &question, true
}

// FindRandomQuestion
func (d *Database) FindRandomQuestion(streamId int) (*Question, error) {
	defaultQuestion := &Question{
		ID:   0,
		Text: "Go ask ChatGPT for your question!",
	}
	rows, err := d.db.Query(FIND_RANDOM_QUESTION, streamId)
	defer func() { _ = rows.Close() }()
	if err != nil {
		log.Println("Error finding random question: ", err)
		return defaultQuestion, err
	}
	if !rows.Next() {
		return defaultQuestion, errors.New("no questions found")
	}

	var question Question
	rows.Scan(&question.ID, &question.Text, &question.Disabled, &question.SkipCount)

	return &question, nil
}

// CreateQuestion
func (d *Database) CreateQuestion(text string) (*Question, error) {
	rows, _ := d.db.Query(FIND_QUESTION_BY_TEXT, text)
	defer func() { _ = rows.Close() }()
	if rows.Next() {
		return nil, errors.New("That question already exists")
	}

	statement, err := d.db.Prepare(INSERT_QUESTION)
	if statement != nil {
		defer func() { _ = statement.Close() }()
	}
	if err != nil {
		log.Println("Error preparing insert question statement: ", err)
		return nil, err
	}

	result, err := statement.Exec(text)
	if err != nil {
		log.Printf("Error updating stream question for question(%s): %x\n", text, err)
		return nil, err
	}

	newID, err := result.LastInsertId()

	return &Question{
		ID:   int(newID),
		Text: text,
	}, nil
}

const question_table string = `
CREATE TABLE IF NOT EXISTS question (
    id INTEGER PRIMARY KEY,
    text TEXT UNIQUE
    )`
