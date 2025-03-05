package timeboost

import (
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

const sqliteFileName = "validated_bids.db?_journal_mode=WAL"

type SqliteDatabase struct {
	sqlDB               *sqlx.DB
	lock                sync.Mutex
	currentTableVersion int
}

func NewDatabase(path string) (*SqliteDatabase, error) {
	//#nosec G304
	if _, err := os.Stat(path); err != nil {
		if err = os.MkdirAll(path, fs.ModeDir); err != nil {
			return nil, err
		}
	}
	filePath := filepath.Join(path, sqliteFileName)
	db, err := sqlx.Open("sqlite3", filePath)
	if err != nil {
		return nil, err
	}
	err = dbInit(db, schemaList)
	if err != nil {
		return nil, err
	}
	return &SqliteDatabase{
		sqlDB:               db,
		currentTableVersion: -1,
	}, nil
}

func dbInit(db *sqlx.DB, schemaList []string) error {
	version, err := fetchVersion(db)
	if err != nil {
		return err
	}
	for index, schema := range schemaList {
		// If the current version is less than the version of the schema, update the database
		if index+1 > version {
			err = executeSchema(db, schema, index+1)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func fetchVersion(db *sqlx.DB) (int, error) {
	flagValue := make([]int, 0)
	// Fetch the current version of the database
	err := db.Select(&flagValue, "SELECT FlagValue FROM Flags WHERE FlagName = 'CurrentVersion'")
	if err != nil {
		if !strings.Contains(err.Error(), "no such table") {
			return 0, err
		}
		// If the table doesn't exist, create it
		_, err = db.Exec(flagSetup)
		if err != nil {
			return 0, err
		}
		// Fetch the current version of the database
		err = db.Select(&flagValue, "SELECT FlagValue FROM Flags WHERE FlagName = 'CurrentVersion'")
		if err != nil {
			return 0, err
		}
	}
	if len(flagValue) > 0 {
		return flagValue[0], nil
	} else {
		return 0, fmt.Errorf("no version found")
	}
}

func executeSchema(db *sqlx.DB, schema string, version int) error {
	// Begin a transaction, so that we update the version and execute the schema atomically
	tx, err := db.Beginx()
	if err != nil {
		return err
	}

	// Execute the schema
	_, err = tx.Exec(schema)
	if err != nil {
		return err
	}
	// Update the version of the database
	_, err = tx.Exec(fmt.Sprintf("UPDATE Flags SET FlagValue = %d WHERE FlagName = 'CurrentVersion'", version))
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (d *SqliteDatabase) InsertBid(b *ValidatedBid) error {
	d.lock.Lock()
	defer d.lock.Unlock()
	query := `INSERT INTO Bids (
        ChainID, Bidder, ExpressLaneController, AuctionContractAddress, Round, Amount, Signature
    ) VALUES (
        :ChainID, :Bidder, :ExpressLaneController, :AuctionContractAddress, :Round, :Amount, :Signature
    )`
	params := map[string]interface{}{
		"ChainID":                b.ChainId.String(),
		"Bidder":                 b.Bidder.Hex(),
		"ExpressLaneController":  b.ExpressLaneController.Hex(),
		"AuctionContractAddress": b.AuctionContractAddress.Hex(),
		"Round":                  b.Round,
		"Amount":                 b.Amount.String(),
		"Signature":              hex.EncodeToString(b.Signature),
	}
	_, err := d.sqlDB.NamedExec(query, params)
	if err != nil {
		return err
	}
	return nil
}

func (d *SqliteDatabase) GetBids(maxDbRows int) ([]*SqliteDatabaseBid, uint64, error) {
	d.lock.Lock()
	defer d.lock.Unlock()
	var maxRound uint64
	query := `SELECT MAX(Round) FROM Bids`
	err := d.sqlDB.Get(&maxRound, query)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to fetch maxRound from bids: %w", err)
	}
	var sqlDBbids []*SqliteDatabaseBid
	if maxDbRows == 0 {
		if err := d.sqlDB.Select(&sqlDBbids, "SELECT * FROM Bids WHERE Round < ? ORDER BY Round ASC", maxRound); err != nil {
			return nil, 0, err
		}
		return sqlDBbids, maxRound, nil
	}
	if err := d.sqlDB.Select(&sqlDBbids, "SELECT * FROM Bids WHERE Round < ? ORDER BY Round ASC LIMIT ?", maxRound, maxDbRows); err != nil {
		return nil, 0, err
	}
	// We should return contiguous set of bids
	for i := len(sqlDBbids) - 1; i > 0; i-- {
		if sqlDBbids[i].Round != sqlDBbids[i-1].Round {
			return sqlDBbids[:i], sqlDBbids[i].Round, nil
		}
	}
	// If we can't determine a contiguous set of bids, we abort and retry again.
	// Saves us from cases where we sometime push same batch data twice
	return nil, 0, nil
}

func (d *SqliteDatabase) DeleteBids(round uint64) error {
	d.lock.Lock()
	defer d.lock.Unlock()
	query := `DELETE FROM Bids WHERE Round < ?`
	_, err := d.sqlDB.Exec(query, round)
	return err
}
