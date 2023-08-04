package db

const INSERT_USER string = `
INSERT INTO user (id, username, displayName)
VALUES (?, ?, ?)
`

const FIND_ALL_USERS string = `
SELECT id, username, displayName
FROM user
`

const FIND_ALL_APIKEY_USERS string = `
SELECT id, username, displayName
FROM user
WHERE apiKey IS NOT NULL
`

const FIND_USER_BY_ID string = `
SELECT id, username, displayName
FROM user
WHERE id=?
`

const FIND_USER_BY_USERNAME string = `
SELECT id, username, displayName
FROM user
WHERE username=?
`

const FIND_USER_BY_APIKEY string = `
SELECT id, username, displayName
FROM user
WHERE apiKey=?
`

const FIND_USER_TIMES_FIRST string = `
SELECT count(id) as timesFirst
FROM stream
WHERE userId=? AND first_userId=?
`

const FIND_TIMES_FIRST_LEADERS string = `
SELECT u.id, u.username, u.displayName, count(u.id) as timesFirst
FROM stream s, user u
WHERE s.first_userId=u.id AND s.userId=?
GROUP BY u.id
ORDER BY timesFirst DESC
LIMIT ?
`

const INSERT_STREAM string = `
INSERT INTO stream (userId, startedAt)
VALUES (?,?)
`

const FIND_STREAM_BY_ID string = `
SELECT id, twid, title, startedAt, endedAt, userId, first_userId, qotdId
FROM stream
WHERE id=?
`

const FIND_CURRENT_STREAM_BY_USERID string = `
SELECT id, twid, title, startedAt, endedAt, userId, first_userId, qotdId
FROM stream
WHERE endedAt IS NULL AND userId=?
LIMIT 1
`

const FIND_ALL_CURRENT_STREAMS string = `
SELECT id, twid, title, startedAt, endedAt, userId, first_userId, qotdId
FROM stream
WHERE endedAt IS NULL
`

const UPDATE_APIKEY_BY_USERID string = `
UPDATE user
SET apiKey=?
WHERE id=?
`

const UPDATE_FIRST_USER string = `
UPDATE stream
SET first_userId=?
WHERE id=?
`

const UPDATE_STREAM_QUESTION string = `
UPDATE stream
SET qotdId=?
WHERE id=?
`

const UPDATE_STREAM_ENDED string = `
UPDATE stream
SET endedAt=?
WHERE id=?
`

const UPDATE_STREAM_INFO string = `
UPDATE stream
SET twid=?, title=?
WHERE id=?
`

const INSERT_QUESTION string = `
INSERT INTO question (text)
VALUES (?)
`

const FIND_QUESTION_BY_ID string = `
SELECT id, text
FROM question
WHERE id=?
`

const FIND_QUESTION_BY_TEXT string = `
SELECT id, text
FROM question
WHERE text=?
`

const FIND_RANDOM_QUESTION string = `
SELECT id, text
FROM question
WHERE NOT EXISTS (SELECT qotdId FROM stream WHERE qotdId=question.id AND userId=?)
ORDER BY RANDOM()
LIMIT 1
`
