package model

import (
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/sprungknoedl/dagobert/pkg/fp"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestRescheduleStaleJobs(t *testing.T) {
	db, close := setupDB()
	defer close()

	kase := Case{ID: fp.Random(10), Name: "Test Case"}
	assert.Nil(t, db.SaveCase(kase))

	stale := Job{ID: fp.Random(10), CaseID: kase.ID, Name: "Hayabusa", Status: "Running"}
	done := Job{ID: fp.Random(10), CaseID: kase.ID, Name: "Hayabusa", Status: "Success"}
	assert.Nil(t, db.PushJob(stale))
	assert.Nil(t, db.PushJob(done))

	// nothing is scheduled yet, so nothing can be popped
	_, err := db.PopJob([]string{"Hayabusa"})
	assert.Equal(t, gorm.ErrRecordNotFound, err)

	// rescheduling makes the stale "Running" job available again
	assert.Nil(t, db.RescheduleStaleJobs())

	job, err := db.PopJob([]string{"Hayabusa"})
	assert.Nil(t, err)
	assert.Equal(t, stale.ID, job.ID)
	assert.Equal(t, "Running", job.Status)

	// the finished job stays untouched
	_, err = db.PopJob([]string{"Hayabusa"})
	assert.Equal(t, gorm.ErrRecordNotFound, err)
}

// The runner pool polls PopJob from multiple goroutines; concurrent write
// statements on a file-backed database must wait for the lock (busy_timeout)
// instead of failing with SQLITE_BUSY.
func TestPopJobConcurrent(t *testing.T) {
	dburl := "file:" + filepath.Join(t.TempDir(), "test.db") + "?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)"
	db, err := Connect(dburl)
	assert.Nil(t, err)
	defer db.RawConn.Close()

	source, _ := iofs.New(Migrations, "migrations")
	driver, _ := sqlite.WithInstance(db.RawConn, &sqlite.Config{})
	m, _ := migrate.NewWithInstance("iofs", source, "sqlite", driver)
	assert.Nil(t, m.Up())

	kase := Case{ID: fp.Random(10), Name: "Test Case"}
	assert.Nil(t, db.SaveCase(kase))
	for range 20 {
		assert.Nil(t, db.PushJob(Job{ID: fp.Random(10), CaseID: kase.ID, Name: "Hayabusa", Status: "Scheduled"}))
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	popped := map[string]bool{}
	for range 4 {
		wg.Go(func() {
			for {
				job, err := db.PopJob([]string{"Hayabusa"})
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return
				}
				if !assert.Nil(t, err) {
					return
				}

				mu.Lock()
				assert.False(t, popped[job.ID], "job %s popped twice", job.ID)
				popped[job.ID] = true
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	assert.Len(t, popped, 20)
}
