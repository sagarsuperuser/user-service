package store

import (
	"sync"

	"github.com/sagarsuperuser/userprofile/internal/common"
)

// Store provides database access to all raw objects.
// Can be used to create cache layer on top of Database.
type Store struct {
	driver       Driver
	userCache    sync.Map
	sessionCache sync.Map
	now          common.NowFunc
}

// New creates a new instance of Store.
func New(driver Driver, now common.NowFunc) *Store {
	return &Store{
		driver: driver,
		now:    now,
	}
}

func (s *Store) Close() error {
	return s.driver.Close()
}
