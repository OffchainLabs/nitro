package db

import (
	"os"

	"github.com/jmoiron/sqlx"
	"github.com/offchainlabs/nitro/timeboost"
)

type Database interface {
	SaveBids(bids []*timeboost.Bid) error
	DeleteBids(round uint64)
}

type BidOption func(b *BidQuery)

type BidQuery struct {
	filters    []string
	args       []interface{}
	startRound int
	endRound   int
}

type Db struct {
	db *sqlx.DB
}

func NewDb(path string) (*Db, error) {
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
	return &Db{
		db: db,
	}, nil
}

func (d *Db) InsertBids(bids []*timeboost.Bid) error {
	for _, b := range bids {
		if err := d.InsertBid(b); err != nil {
			return err
		}
	}
	return nil
}

func (d *Db) InsertBid(b *timeboost.Bid) error {
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
	_, err := d.db.NamedExec(query, params)
	if err != nil {
		return err
	}
	return nil
}

func (d *Db) DeleteBids(round uint64) error {
	query := `DELETE FROM Bids WHERE Round < ?`
	_, err := d.db.Exec(query, round)
	return err
}
