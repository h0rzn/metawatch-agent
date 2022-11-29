package db

const (
	URI    string = "mongodb://"
	DBName string = "db1"
)

type DB struct {
	Client *Client
}

// Validate checks if database is valid and usable
// empty = true means the database is unused
func (db *DB) Validate() (empty bool, err error) {
	// check if collections exist

	return false, nil
}

// InitScheme initiates collections
func (db *DB) InitScheme() {

}
