package db

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/offchainlabs/nitro/timeboost"
)

type Database interface {
	SaveBids(bids []*timeboost.Bid) error
	DeleteBids(round uint64)
}

type SqliteDatabase struct {
	sqlDB               *sqlx.DB
	lock                sync.Mutex
	currentTableVersion int
}

func NewDatabase(path string) (*SqliteDatabase, error) {
	//#nosec G304
	if _, err := os.Stat(path); err != nil {
		_, err = os.Create(path)
		if err != nil {
			return nil, err
		}
	}
	db, err := sqlx.Open("sqlite3", path)
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

func (d *SqliteDatabase) InsertBids(bids []*timeboost.Bid) error {
	for _, b := range bids {
		if err := d.InsertBid(b); err != nil {
			return err
		}
	}
	return nil
}

func (d *SqliteDatabase) InsertBid(b *timeboost.Bid) error {
	d.lock.Lock()
	defer d.lock.Unlock()
	query := `INSERT INTO Bids (
        ChainID, ExpressLaneController, AuctionContractAddress, Round, Amount, Signature
    ) VALUES (
        :ChainID, :ExpressLaneController, :AuctionContractAddress, :Round, :Amount, :Signature
    )`
	params := map[string]interface{}{
		"ChainID":                b.ChainId.String(),
		"ExpressLaneController":  b.ExpressLaneController.Hex(),
		"AuctionContractAddress": b.AuctionContractAddress.Hex(),
		"Round":                  b.Round,
		"Amount":                 b.Amount.String(),
		"Signature":              b.Signature,
	}
	_, err := d.sqlDB.NamedExec(query, params)
	if err != nil {
		return err
	}
	return nil
}

func (d *SqliteDatabase) DeleteBids(round uint64) error {
	d.lock.Lock()
	defer d.lock.Unlock()
	query := `DELETE FROM Bids WHERE Round < ?`
	_, err := d.sqlDB.Exec(query, round)
	return err
}
