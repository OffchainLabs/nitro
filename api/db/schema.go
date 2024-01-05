package db

var (
	//nolint:unused
	schema = `
CREATE TABLE Challenges (
    AssertionHash TEXT NOT NULL PRIMARY KEY,
    UNIQUE(AssertionHash)
);

CREATE TABLE Edges (
    ID TEXT NOT NULL PRIMARY KEY,
    ChallengeLevel INTEGER,
    OriginId TEXT NOT NULL,
    StartHistoryRoot TEXT NOT NULL,
    StartHeight INTEGER NOT NULL,
    EndHistoryRoot TEXT NOT NULL,
    EndHeight INTEGER NOT NULL,
    CreatedAtBlock INTEGER NOT NULL,
    MutualID TEXT NOT NULL,
    ClaimID TEXT,
    HasChildren BOOLEAN NOT NULL,
    LowerChildID TEXT NOT NULL,
    UpperChildID TEXT NOT NULL,
    MiniStaker TEXT,
    AssertionHash TEXT NOT NULL,
    HasRival BOOLEAN,
    Status TEXT NOT NULL,
    HasLengthOneRival BOOLEAN,
    FOREIGN KEY(LowerChildID) REFERENCES Edges(Id),
    FOREIGN KEY(UpperChildID) REFERENCES Edges(Id),
    FOREIGN KEY(AssertionHash) REFERENCES Challenges(AssertionHash)
);

CREATE TABLE Assertions (
    Hash TEXT NOT NULL PRIMARY KEY,
    ConfirmPeriodBlocks INTEGER,
    RequiredStake TEXT NOT NULL,
    ParentAssertionHash TEXT NOT NULL,
    InboxMaxCount TEXT NOT NULL,
    AfterInboxBatchAcc TEXT NOT NULL,
    WasmModuleRoot TEXT NOT NULL,
    ChallengeManager TEXT NOT NULL,
    CreationBlock INTEGER,
    TransactionHash TEXT NOT NULL,
    BeforeStateBlockHash TEXT NOT NULL,
    BeforeStateSendRoot TEXT NOT NULL,
    BeforeStateMachineStatus TEXT NOT NULL,
    AfterStateBlockHash TEXT NOT NULL,
    AfterStateSendRoot TEXT NOT NULL,
    AfterStateMachineStatus TEXT NOT NULL,
    FirstChildBlock INTEGER,
    SecondChildBlock INTEGER,
    IsFirstChild BOOLEAN,
    Status TEXT NOT NULL,
    ConfigHash TEXT NOT NULL,
    FOREIGN KEY(Hash) REFERENCES Challenges(AssertionHash),
    FOREIGN KEY(ParentAssertionHash) REFERENCES Assertions(Hash)
);

CREATE INDEX idx_edge_assertion ON Edge(AssertionHash);
CREATE INDEX idx_assertions_assertion ON Assertions(AssertionHash);
CREATE INDEX idx_edge_claim_id ON Edge(ClaimID);
CREATE INDEX idx_edge_end_height ON Edge(EndHeight);
CREATE INDEX idx_edge_end_history_root ON Edge(EndHistoryRoot);
`
)
