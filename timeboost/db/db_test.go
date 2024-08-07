package db

import (
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
	"github.com/offchainlabs/nitro/timeboost"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type DatabaseBid struct {
	Id                     uint64 `db:"Id"`
	ChainId                string `db:"ChainId"`
	ExpressLaneController  string `db:"ExpressLaneController"`
	AuctionContractAddress string `db:"AuctionContractAddress"`
	Round                  uint64 `db:"Round"`
	Amount                 string `db:"Amount"`
	Signature              string `db:"Signature"`
}

func TestInsertAndFetchBids(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "*")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(tmpDir))
	})
	tmpFile := filepath.Join(tmpDir, "database.sql?_journal_mode=WAL")
	db, err := NewDatabase(tmpFile)
	require.NoError(t, err)

	bids := []*timeboost.Bid{
		{
			ChainId:                big.NewInt(1),
			ExpressLaneController:  common.HexToAddress("0x0000000000000000000000000000000000000001"),
			AuctionContractAddress: common.HexToAddress("0x0000000000000000000000000000000000000002"),
			Round:                  1,
			Amount:                 big.NewInt(100),
			Signature:              []byte("signature1"),
		},
		{
			ChainId:                big.NewInt(2),
			ExpressLaneController:  common.HexToAddress("0x0000000000000000000000000000000000000003"),
			AuctionContractAddress: common.HexToAddress("0x0000000000000000000000000000000000000004"),
			Round:                  2,
			Amount:                 big.NewInt(200),
			Signature:              []byte("signature2"),
		},
	}
	require.NoError(t, db.InsertBids(bids))
	gotBids := make([]*DatabaseBid, 2)
	err = db.sqlDB.Select(&gotBids, "SELECT * FROM Bids ORDER BY Id")
	require.NoError(t, err)
	require.Equal(t, bids[0].Amount.String(), gotBids[0].Amount)
	require.Equal(t, bids[1].Amount.String(), gotBids[1].Amount)
}

func TestInsertBids(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")

	d := &SqliteDatabase{sqlDB: sqlxDB, currentTableVersion: -1}

	bids := []*timeboost.Bid{
		{
			ChainId:                big.NewInt(1),
			ExpressLaneController:  common.HexToAddress("0x0000000000000000000000000000000000000001"),
			AuctionContractAddress: common.HexToAddress("0x0000000000000000000000000000000000000002"),
			Round:                  1,
			Amount:                 big.NewInt(100),
			Signature:              []byte("signature1"),
		},
		{
			ChainId:                big.NewInt(2),
			ExpressLaneController:  common.HexToAddress("0x0000000000000000000000000000000000000003"),
			AuctionContractAddress: common.HexToAddress("0x0000000000000000000000000000000000000004"),
			Round:                  2,
			Amount:                 big.NewInt(200),
			Signature:              []byte("signature2"),
		},
	}

	for _, bid := range bids {
		mock.ExpectExec("INSERT INTO Bids").WithArgs(
			bid.ChainId.String(),
			bid.ExpressLaneController.Hex(),
			bid.AuctionContractAddress.Hex(),
			bid.Round,
			bid.Amount.String(),
			bid.Signature,
		).WillReturnResult(sqlmock.NewResult(1, 1))
	}

	err = d.InsertBids(bids)
	assert.NoError(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestDeleteBidsLowerThanRound(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")

	d := &SqliteDatabase{
		sqlDB:               sqlxDB,
		currentTableVersion: -1,
	}

	round := uint64(10)

	mock.ExpectExec("DELETE FROM Bids WHERE Round < ?").
		WithArgs(round).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = d.DeleteBids(round)
	assert.NoError(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}
