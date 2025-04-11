package livedbsnapshotter

import (
	"context"
	"sync/atomic"

	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type Config struct {
	Enable bool   `koanf:"enable"`
	Dir    string `koanf:"dir"`
}

var DefaultConfig = Config{
	Enable: false,
	Dir:    "",
}

func ConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultConfig.Enable, "enable creation of live db snapshots")
	f.String(prefix+".dir", DefaultConfig.Dir, "path to the directory for saving db snapshots")
}

type LiveDBSnapshotter struct {
	stopwaiter.StopWaiter
	db             ethdb.Database
	dbName         string
	trigger        chan struct{}
	dir            string
	isSnapshotDue  atomic.Bool
	withScheduling bool

	chainedTrigger chan struct{}
}

func NewLiveDBSnapshotter(db ethdb.Database, dbName string, trigger chan struct{}, dir string, withScheduling bool, chainedTrigger chan struct{}) *LiveDBSnapshotter {
	return &LiveDBSnapshotter{
		db:             db,
		dbName:         dbName,
		trigger:        trigger,
		dir:            dir,
		withScheduling: withScheduling,
		chainedTrigger: chainedTrigger,
	}
}

func (l *LiveDBSnapshotter) Start(ctx context.Context) {
	l.StopWaiter.Start(ctx, l)
	l.LaunchThread(l.scheduleSnapshotCreation)
}

func (l *LiveDBSnapshotter) scheduleSnapshotCreation(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-l.trigger:
			log.Info("Live databases snapshot creation scheduled", "databases", l.dbName)
		}

		if l.dir == "" {
			log.Error("Aborting live databases snapshot creation as destination directory is empty")
			continue
		}
		l.isSnapshotDue.Store(true)
		if !l.withScheduling {
			l.CreateDBSnapshotIfDue()
		}
	}
}

func (l *LiveDBSnapshotter) CreateDBSnapshotIfDue() {
	if l.Stopped() || !l.isSnapshotDue.Load() {
		return
	}

	log.Info("Beginning snapshot creation", "databases", l.dbName)
	if err := l.db.CreateDBSnapshot(l.dir); err != nil {
		log.Error("Snapshot creation for database failed", "databases", l.dbName, "err", err)
	} else {
		log.Info("Live snapshot was successfully created", "databases", l.dbName)
	}
	l.isSnapshotDue.Store(false)

	// As snapshot of consensus can only be taken after execution's is done
	if l.chainedTrigger != nil {
		select {
		case l.chainedTrigger <- struct{}{}:
		default:
		}
	}
}
