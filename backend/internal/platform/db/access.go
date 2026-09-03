package db

// DBTX exposes the query executor used by sqlc so higher-level persistence
// adapters can start a transaction without modifying generated db.go.
func (q *Queries) DBTX() DBTX {
	return q.db
}
