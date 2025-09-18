// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

package db

var (
	// flagSetup is the initial setup for the flags table.
	// It creates the table if it doesn't exist and sets the CurrentVersion to 0.
	flagSetup = `
CREATE TABLE IF NOT EXISTS Flags (
    FlagName TEXT NOT NULL PRIMARY KEY,
    FlagValue INTEGER NOT NULL
);
INSERT INTO Flags (FlagName, FlagValue) VALUES ('CurrentVersion', 0);
`
	// schemaList is a list of schema versions.
	// The first element is the initial schema,
	// and each subsequent element is a migration from the previous version to the new version.
	version1 = `
CREATE TABLE IF NOT EXISTS Challenges (
    Hash TEXT NOT NULL PRIMARY KEY,
    UNIQUE(Hash)
);

CREATE TABLE IF NOT EXISTS EdgeClaims (
    ClaimId TEXT NOT NULL PRIMARY KEY,
    RefersTo TEXT NOT NULL, -- 'edge' or 'assertion'
    FOREIGN KEY(ClaimId) REFERENCES Edges(Id),
    FOREIGN KEY(ClaimId) REFERENCES Assertions(Hash)
);

CREATE TABLE IF NOT EXISTS Edges (
    Id TEXT NOT NULL PRIMARY KEY,
    ChallengeLevel INTEGER NOT NULL,
    OriginId TEXT NOT NULL,
    StartHistoryRoot TEXT NOT NULL,
    StartHeight INTEGER NOT NULL,
    EndHistoryRoot TEXT NOT NULL,
    EndHeight INTEGER NOT NULL,
    CreatedAtBlock INTEGER NOT NULL,
    MutualId TEXT NOT NULL,
    ClaimId TEXT NOT NULL,
    MiniStaker TEXT NOT NULL,
    AssertionHash TEXT NOT NULL,
    HasChildren BOOLEAN NOT NULL,
    LowerChildId TEXT NOT NULL,
    UpperChildId TEXT NOT NULL,
    HasRival BOOLEAN NOT NULL,
    Status TEXT NOT NULL,
    HasLengthOneRival BOOLEAN NOT NULL,
    IsRoyal BOOLEAN NOT NULL,
    RawAncestors TEXT NOT NULL,
    LastUpdatedAt DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(LowerChildID) REFERENCES Edges(Id),
    FOREIGN KEY(ClaimId) REFERENCES EdgeClaims(ClaimId),
    FOREIGN KEY(UpperChildID) REFERENCES Edges(Id),
    FOREIGN KEY(AssertionHash) REFERENCES Challenges(Hash)
);

CREATE TABLE IF NOT EXISTS Assertions (
    Hash TEXT NOT NULL PRIMARY KEY,
    ConfirmPeriodBlocks INTEGER NOT NULL,
    RequiredStake TEXT NOT NULL,
    ParentAssertionHash TEXT NOT NULL,
    InboxMaxCount TEXT NOT NULL,
    AfterInboxBatchAcc TEXT NOT NULL,
    WasmModuleRoot TEXT NOT NULL,
    ChallengeManager TEXT NOT NULL,
    CreationBlock INTEGER NOT NULL,
    TransactionHash TEXT NOT NULL,
    BeforeStateBlockHash TEXT NOT NULL,
    BeforeStateSendRoot TEXT NOT NULL,
    BeforeStateBatch INTEGER NOT NULL,
    BeforeStatePosInBatch INTEGER NOT NULL,
    BeforeStateMachineStatus INTEGER NOT NULL,
    AfterStateBlockHash TEXT NOT NULL,
    AfterStateSendRoot TEXT NOT NULL,
    AfterStateBatch INTEGER NOT NULL,
    AfterStatePosInBatch INTEGER NOT NULL,
    AfterStateMachineStatus INTEGER NOT NULL,
    FirstChildBlock INTEGER,
    SecondChildBlock INTEGER,
    IsFirstChild BOOLEAN NOT NULL,
    Status TEXT NOT NULL,
    LastUpdatedAt DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(Hash) REFERENCES Challenges(Hash),
    FOREIGN KEY(ParentAssertionHash) REFERENCES Assertions(Hash)
);

CREATE INDEX IF NOT EXISTS idx_edge_assertion ON Edges(AssertionHash);
CREATE INDEX IF NOT EXISTS idx_assertions_assertion ON Assertions(Hash);
CREATE INDEX IF NOT EXISTS idx_edge_claim_id ON Edges(ClaimId);
CREATE INDEX IF NOT EXISTS idx_edge_end_height ON Edges(EndHeight);
CREATE INDEX IF NOT EXISTS idx_edge_end_history_root ON Edges(EndHistoryRoot);

CREATE TRIGGER IF NOT EXISTS UpdateEdgeTimestamp
AFTER UPDATE ON Edges
FOR EACH ROW
BEGIN
   UPDATE Edges SET LastUpdatedAt = CURRENT_TIMESTAMP WHERE Id = NEW.Id;
END;

CREATE TRIGGER IF NOT EXISTS UpdateAssertionTimestamp
AFTER UPDATE ON Assertions
FOR EACH ROW
BEGIN
   UPDATE Assertions SET LastUpdatedAt = CURRENT_TIMESTAMP WHERE Hash = NEW.Hash;
END;
`
	version2 = `
ALTER TABLE Edges ADD COLUMN InheritedTimer INTEGER NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS CollectMachineHashes (
    WasmModuleRoot TEXT NOT NULL,
    FromBatch INTEGER NOT NULL,
    PositionInBatch INTEGER NOT NULL,
    BatchLimit INTEGER NOT NULL,
    BlockChallengeHeight INTEGER NOT NULL,
    RawStepHeights TEXT NOT NULL,
    NumDesiredHashes INTEGER NOT NULL,
    MachineStartIndex INTEGER NOT NULL,
    StepSize INTEGER NOT NULL,
    StartTime DATETIME NOT NULL,
    FinishTime DATETIME
);
`
	version3 = `
	ALTER TABLE Edges ADD COLUMN CumulativePathTimer INTEGER NOT NULL DEFAULT 0;
`
	// schemaList is a list of schema versions.
	schemaList = []string{version1, version2, version3}
)
