package db

var (
	//nolint:unused
	schema = `
CREATE TABLE Challenges (
    Hash TEXT NOT NULL PRIMARY KEY,
    UNIQUE(Hash)
);

CREATE TABLE EdgeClaims (
    ClaimId TEXT NOT NULL PRIMARY KEY,
    RefersTo TEXT NOT NULL, -- 'edge' or 'assertion'
    FOREIGN KEY(ClaimId) REFERENCES Edges(Id),
    FOREIGN KEY(ClaimId) REFERENCES Assertions(Hash)
);

CREATE TABLE Edges (
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
    CumulativePathTimer INTEGER NOT NULL,
    LastUpdatedAt DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(LowerChildID) REFERENCES Edges(Id),
    FOREIGN KEY(ClaimId) REFERENCES EdgeClaims(ClaimId),
    FOREIGN KEY(UpperChildID) REFERENCES Edges(Id),
    FOREIGN KEY(AssertionHash) REFERENCES Challenges(Hash)
);

CREATE TABLE Assertions (
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

CREATE INDEX idx_edge_assertion ON Edges(AssertionHash);
CREATE INDEX idx_assertions_assertion ON Assertions(Hash);
CREATE INDEX idx_edge_claim_id ON Edges(ClaimId);
CREATE INDEX idx_edge_end_height ON Edges(EndHeight);
CREATE INDEX idx_edge_end_history_root ON Edges(EndHistoryRoot);

CREATE TRIGGER UpdateEdgeTimestamp
AFTER UPDATE ON Edges
FOR EACH ROW
BEGIN
   UPDATE Edges SET LastUpdatedAt = CURRENT_TIMESTAMP WHERE Id = NEW.Id;
END;

CREATE TRIGGER UpdateAssertionTimestamp
AFTER UPDATE ON Assertions
FOR EACH ROW
BEGIN
   UPDATE Assertions SET LastUpdatedAt = CURRENT_TIMESTAMP WHERE Hash = NEW.Hash;
END;
`
)
