package api

func (a *API) authorizeToken(token string) bool {
	dbToken := &Token{}
	result := a.db.Where("token = ?", token).First(dbToken)

	return result.RowsAffected > 0
}
