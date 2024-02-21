package db

const INSERT_USER string = `
INSERT INTO user (id, username, displayName)
VALUES (?, ?, ?)
`

const UPDATE_USERNAME string = `
UPDATE user
SET username=?, displayName=?
WHERE id=?
`

const FIND_ALL_USERS string = `
SELECT id, username, displayName
FROM user
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
SELECT u.id, u.username, u.displayName
FROM user u, stream_config sc
WHERE sc.apiKey=? AND u.id=sc.userId
`

const FIND_USER_TIMES_FIRST string = `
SELECT count(s.id) as timesFirst
FROM stream s, stream_config sc
WHERE s.userId=sc.userId AND s.userId=? AND s.first_userId=?
    AND (s.endedAt>sc.firstEpoch OR s.endedAt IS NULL)
`

const FIND_USER_TIMES_FIRST_ALL string = `
SELECT count(s.id) as timesFirst
FROM stream s
WHERE s.userId=? AND s.first_userId=?
`

const FIND_TIMES_FIRST_LEADERS string = `
SELECT u.id, u.username, u.displayName, count(u.id) as timesFirst
FROM stream s, stream_config sc, user u
WHERE s.first_userId=u.id AND s.userId=sc.userId AND s.userId=?
    AND (s.endedAt>sc.firstEpoch OR s.endedAt IS NULL)
GROUP BY u.id
ORDER BY timesFirst DESC
LIMIT ?
`

const FIND_TIMES_FIRST_LEADERS_ALL string = `
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
UPDATE stream_config
SET apiKey=?
WHERE userId=?
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
SELECT id, text, disabled, skipCount
FROM question
WHERE id=?
`

const FIND_QUESTION_BY_TEXT string = `
SELECT id, text, disabled, skipCount
FROM question
WHERE text=?
`

const FIND_RANDOM_QUESTION string = `
SELECT id, text, disabled, skipCount
FROM question
WHERE NOT EXISTS (SELECT qotdId FROM stream WHERE qotdId=question.id AND userId=?)
    AND disabled = false
ORDER BY RANDOM()
LIMIT 1
`

const DISABLE_QUESTION string = `
UPDATE question
SET disabled = true
WHERE id=?
`

const INCREMENT_QUESTION_SKIP string = `
UPDATE question
SET skipCount = skipCount + 1
WHERE id=?
`

const CREATE_STREAM_CONFIG string = `
INSERT INTO stream_config (
    userId,
    botDisabled,
    firstEnabled,
    firstEpoch,
    qotdEnabled,
    qotdEpoch,
    dateUpdated,
    apiKey,
    twitchAuthToken,
    twitchRefreshToken
    )
VALUES(?,?,?,?,?,?,?,?,?,?)
`

const UPDATE_FIRST_EPOCH string = `
UPDATE stream_config
SET firstEpoch=?
WHERE userId=?
`

const UPDATE_TWITCHAUTH_BY_USERID string = `
UPDATE stream_config
SET twitchAuthToken=?, twitchRefreshToken=?
WHERE userId=?
`

const FIND_STREAM_USER_BY_USERID string = `
SELECT u.id, u.username, u.displayName, sc.id, sc.userId, sc.botDisabled, sc.firstEnabled, sc.firstEpoch, sc.qotdEnabled, sc.qotdEpoch, sc.dateUpdated, sc.apiKey, sc.twitchAuthToken, sc.twitchRefreshToken
FROM user u, stream_config sc
WHERE u.id = sc.userId AND userId=?
`

const FIND_STREAM_USER_BY_USERNAME string = `
SELECT u.id, u.username, u.displayName, sc.id, sc.userId, sc.botDisabled, sc.firstEnabled, sc.firstEpoch, sc.qotdEnabled, sc.qotdEpoch, sc.dateUpdated, sc.apiKey, sc.twitchAuthToken, sc.twitchRefreshToken
FROM user u, stream_config sc
WHERE u.id = sc.userId AND u.username=?
`

const FIND_ALL_STREAM_USERS string = `
SELECT u.id, u.username, u.displayName, sc.id, sc.userId, sc.botDisabled, sc.firstEnabled, sc.firstEpoch, sc.qotdEnabled, sc.qotdEpoch, sc.dateUpdated, sc.apiKey, sc.twitchAuthToken, sc.twitchRefreshToken
FROM user u, stream_config sc
WHERE u.id = sc.userId
`
