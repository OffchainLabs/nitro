package db

var (
	flagSetup = `
CREATE TABLE IF NOT EXISTS Flags (
    FlagName TEXT NOT NULL PRIMARY KEY,
    FlagValue INTEGER NOT NULL
);
INSERT INTO Flags (FlagName, FlagValue) VALUES ('CurrentVersion', 0);
`
	version1 = `
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
`
)
