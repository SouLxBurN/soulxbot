package api

// Question is a struct that represents a question in the database
type Question struct {
	ID   int    `json:"id"`
	Text string `json:"text"`
}
