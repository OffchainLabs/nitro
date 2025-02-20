package timeboost

var (
	flagSetup = `
CREATE TABLE IF NOT EXISTS Flags (
    FlagName TEXT NOT NULL PRIMARY KEY,
    FlagValue INTEGER NOT NULL
);
INSERT INTO Flags (FlagName, FlagValue) VALUES ('CurrentVersion', 0);
`
	version1 = `
CREATE TABLE IF NOT EXISTS Bids (
    Id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    ChainId TEXT NOT NULL,
    Bidder TEXT NOT NULL,
    ExpressLaneController TEXT NOT NULL,
    AuctionContractAddress TEXT NOT NULL,
    Round INTEGER NOT NULL,
    Amount TEXT NOT NULL,
    Signature TEXT NOT NULL
);
CREATE INDEX idx_bids_round ON Bids(Round);
`
	schemaList = []string{version1}
)
