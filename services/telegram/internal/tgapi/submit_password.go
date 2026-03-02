package tgapi

func (c *tgCli) SubmitPassword(sessionID string, password string) error {
	return c.authManager.sendPassword(sessionID, password)

}
