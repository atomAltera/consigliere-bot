package bot

// RegisterCommands sets up all bot commands with admin-only middleware
// Middleware order matters: HandleErrors must be outermost to catch all errors,
// RateLimit should run early to drop excessive requests,
// DeleteCommand should run before AdminOnly so commands are always deleted,
// LogCommand should run after AdminOnly to only log authorized commands.
func (b *Bot) RegisterCommands() {
	adminGroup := b.bot.Group()
	adminGroup.Use(b.HandleErrors())
	adminGroup.Use(b.ResolveClub())
	adminGroup.Use(b.RateLimit())
	adminGroup.Use(b.DeleteCommand())
	adminGroup.Use(b.ClubAdminOnly())
	adminGroup.Use(b.LogCommand())

	adminGroup.Handle("/poll", b.handlePoll)
	adminGroup.Handle("/results", b.handleResults)
	adminGroup.Handle("/pin", b.handlePin)
	adminGroup.Handle("/cancel", b.handleCancel)
	adminGroup.Handle("/restore", b.handleRestore)
	adminGroup.Handle("/vote", b.handleVote)
	adminGroup.Handle("/nick", b.handleNick)
	adminGroup.Handle("/call", b.handleCall)
	adminGroup.Handle("/done", b.handleDone)
	adminGroup.Handle("/refresh", b.handleRefresh)
	adminGroup.Handle("/help", b.handleHelp)
}
