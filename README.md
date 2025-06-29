# Multi-purpose, multi-platform bot for Apex Corse

King is a Telegram/Discord bot made to help Apex Corse's departments in their everyday work. It's entirely built with Go, using the correspondent APIs for Telegram and Discord, and with Turso as DB.

## Configuration

### Notifications on push

King's primary functionality is to inform people working in a repo that someone else has pushed changes on remote. This allows the team to always stay coordinated, without any effort. King accomplishes this with the support of Apex Corse's [`notify-push` action](https://github.com/Formula-SAE/notify-push).

To configure this behaviour, you have to follow this steps:

1. Add King in a Telegram chat/Discord channel.
2. Obtain the chat/channel id where you installed King and save it.
3. Add `notify-push` to the repository you want to track, using [this template](https://github.com/Formula-SAE/king/blob/main/.github/workflows/notify.yaml).
4. Create an API token via the King API.
5. Configure `API_URL`, `API_TOKEN` and `PROVIDERS` repository secrets.

Now King will track the changes happening on your repo and will notify you on your chat/channel.
