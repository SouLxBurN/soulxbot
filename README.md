# SouLxBot Chatbot
Twitch Chatbot that provides a first user to chat after going live feature, and an automated question of the day using `!qotd`.

Lots of feature still to be developed and polished, this is still quite the _Work In Progress_.

## Setup

### Copy `.env.template` as `.env`
Create a copy of of the environment variable template.
You will add values to this new file as you proceed through these instructions.

### Register a Twitch App
If you already have a Twitch app that you can use, you may skip this step.

Follow the steps outlined at [Registering Your App](https://dev.twitch.tv/docs/authentication/register-app) to register a new twitch app.

### Create OAuth Token for Bot User
The quick and dirty way is to use a tool like https://twitchapps.com/tmi to create an OAuth token for the chatbot user.
Otherwise, you'll need to follow a similar flow to below, with different chat related permissions.
[Twitch Authenticate Bot](https://dev.twitch.tv/docs/irc/authenticate-bot) will get you started.

Add the username for the bot, and the OAuth token you generated to your `.env`.
```
SOULXBOT_USER=
SOULXBOT_OAUTH=
```

### Client ID & Secret
Add your Client ID and Client Secret from your created Twitch App into `.env`.
```
SOULXBOT_CLIENTID=
SOULXBOT_CLIENTSECRET=
```

### Get User Code
In a browser goto the following, replacing the `{client_id}` with your app's `client_id`.
You will want to auth with the twitch channel you want to test with.
The `channel:manage:predictions` is needed for the dice game predictions.
```
https://id.twitch.tv/oauth2/authorize?client_id={client_id}&redirect_uri=http%3A%2F%2Flocalhost&response_type=code&scope=channel%3Amanage%3Apredictions
```

This will redirect you to your app's redirect uri, and the redirect url will look similar to:
```
http://localhost/?code={user_code}&scope=channel%3Amanage%3Apredictions
```

Extract the `{user_code}` the code that is returned. You will need it in the next step.

### Get User Authorization Token
Send a `POST` request to the URL below, replacing `{client_id}`, `{client_secret}`, and `{user_code}` with the code you just obtained above.

```
POST https://id.twitch.tv/oauth2/token?client_id={client_id}&client_secret={client_secret}&code={user_code}&grant_type=authorization_code&redirect_uri=http%3A%2F%2Flocalhost
```

You should get a response similar to:
```json
{
  "access_token": "abcd1234",
  "expires_in": 14943,
  "refresh_token": "123456789abcdefgh",
  "scope": [ "channel:manage:predictions" ],
  "token_type": "bearer"
}
```

Extract the `access_token` and `refresh_token`, then add them to your `.env` file
```
SOULXBOT_AUTHTOKEN=
SOULXBOT_REFRESHTOKEN=
```

### Running The Bot
- Start the bot locally by running `go run .`
- The bot has a web server running on port `8080`.
    - Use `http://localhost:8080/register?username={stream_user}`, to register a user to have the bot join that stream's channel.
        This will return an API key in order to inform the bot that the user has gone live.
    - ⚠️ Currently, the bot will need to see the user say something in chat somewhere before it can successfully register them. _(Bug 9/6/23)_
    - ⚠️ After registering, you will need to restart the bot in order for it to join the newly registered user's channel. _(Bug 9/6/23)_
    - Inform the bot that a stream has gone live! `http://localhost:8080/golive?key={api_key}` with the API key returned from the `/register` endpoint.

