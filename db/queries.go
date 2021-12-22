package db

const INSERT_USER string = `
INSERT INTO user (username)
VALUES (?)
`

const INCREMENT_TIMES_FIRST string = `
UPDATE user
SET timesFirst = timesFirst + 1
WHERE id=?
`

const FIND_ALL_USERS string = `
SELECT id, username, timesFirst
FROM user
`

const FIND_USER_BY_ID string = `
SELECT id, username, timesFirst
FROM user
WHERE id=?
`

const FIND_USER_BY_USERNAME string = `
SELECT id, username, timesFirst
FROM user
WHERE username=?
`

const FIND_TIMES_FIRST_LEADERS string = `
SELECT id, username, timesFirst
FROM user
ORDER BY timesFirst DESC
LIMIT ?
`

const INSERT_STREAM string = `
INSERT INTO stream (title, startedAt, first_userId)
VALUES (?,?,?)
`

const UPDATE_FIRST_USER string = `
UPDATE stream
SET first_userId=?
WHERE id=?
`
