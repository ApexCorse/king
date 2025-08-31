package discord

func (b *DiscordBot) sendPrivateMessage(userID string, message string) error {
	channel, err := b.session.UserChannelCreate(userID)
	if err != nil {
		return err
	}

	_, err = b.session.ChannelMessageSend(channel.ID, message)
	if err != nil {
		return err
	}

	return nil
}
