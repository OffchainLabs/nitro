package api

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
)

type Database struct {
	sqlDB               *sqlx.DB
	tableName           string
	currentTableVersion int
	updateInterval      time.Duration
	edges               EdgesProvider
	lastUpdated         time.Time

	versionMutex *sync.RWMutex
	updateMutex  *sync.Mutex
}

type DBConfig struct {
	Enable           bool
	DBPath           string
	TableName        string
	DBUpdateInterval time.Duration
}

func NewDatabase(config *DBConfig, edges EdgesProvider) (*Database, error) {
	if _, err := os.Stat(config.DBPath); err != nil {
		_, err = os.Create(config.DBPath)
		if err != nil {
			return nil, err
		}
	}
	db, err := sqlx.Open("sqlite3", config.DBPath)
	if err != nil {
		return nil, err
	}
	return &Database{
		sqlDB:               db,
		tableName:           config.TableName,
		currentTableVersion: -1,
		updateInterval:      config.DBUpdateInterval,
		edges:               edges,
	}, nil
}

// Start starts the database update loop.
// Update the database every updateInterval.
// Create a new table version after fetching the edges.
// Drops the previous table version after updating.
func (d *Database) Start(ctx context.Context) {
	for {
		err := d.Update(ctx)
		if err != nil {
			log.Error("failed to update database", "err", err)
		}
		time.Sleep(d.updateInterval)
	}
}

func (d *Database) Update(ctx context.Context) error {
	// Only allow one update at a time
	d.updateMutex.Lock()
	defer d.updateMutex.Unlock()
	// TODO: Fetch edges in a stepwise fashion,
	//  like fetching 10 edges at a time and
	//  adding them to the database before fetching the next 10.
	specEdges, err := d.edges.GetEdges(ctx)
	if err != nil {
		return err
	}
	if len(specEdges) != 0 {
		d.versionMutex.RLock()
		currentTableVersion := d.currentTableVersion
		d.versionMutex.RUnlock()
		return d.insertEdges(ctx, specEdges, currentTableVersion)
	}
	return nil
}

func (d *Database) insertEdges(ctx context.Context, specEdges []protocol.SpecEdge, currentTableVersion int) error {
	// Create a new table version
	_, err := d.sqlDB.Exec(
		"CREATE TABLE " + d.tableName + strconv.Itoa(currentTableVersion+1) + " (" +
			"id BLOB PRIMARY KEY," +
			"type TEXT," +
			"startCommitmentHash BLOB," +
			"startCommitmentHeight INTEGER," +
			"endCommitmentHash BLOB," +
			"endCommitmentHeight INTEGER," +
			"createdAtBlock INTEGER," +
			"mutualId BLOB," +
			"originId BLOB," +
			"claimId BLOB," +
			"hasChildren BOOLEAN," +
			"lowerChildId BLOB," +
			"upperChildId BLOB," +
			"miniStaker BLOB," +
			"assertionHash BLOB," +
			"timeUnrivaled INTEGER," +
			"hasRival BOOLEAN," +
			"status TEXT," +
			"hasLengthOneRival BOOLEAN," +
			"topLevelClaimHeight BLOB," +
			"cumulativePathTimer INTEGER" +
			")")
	if err != nil {
		log.Error("failed to create table", "err", err)
		return err
	}

	edges, err := convertSpecEdgeEdgesToEdges(ctx, specEdges, d.edges)
	if err != nil {
		log.Error("failed to convert edges", "err", err)
		return err
	}

	// Insert edges into the new table version
	_, err = d.sqlDB.NamedExec(
		"INSERT INTO "+d.tableName+strconv.Itoa(currentTableVersion+1)+" ("+
			"id,"+
			"type,"+
			"startCommitmentHash,"+
			"startCommitmentHeight,"+
			"endCommitmentHash,"+
			"endCommitmentHeight,"+
			"createdAtBlock,"+
			"mutualId,"+
			"originId,"+
			"claimId,"+
			"hasChildren,"+
			"lowerChildId,"+
			"upperChildId,"+
			"miniStaker,"+
			"assertionHash,"+
			"timeUnrivaled,"+
			"hasRival,"+
			"status,"+
			"hasLengthOneRival,"+
			"topLevelClaimHeight,"+
			"cumulativePathTimer"+
			") VALUES ("+
			":id,"+
			":type,"+
			":startCommitmentHash,"+
			":startCommitmentHeight,"+
			":endCommitmentHash,"+
			":endCommitmentHeight,"+
			":createdAtBlock,"+
			":mutualId,"+
			":originId,"+
			":claimId,"+
			":hasChildren,"+
			":lowerChildId,"+
			":upperChildId,"+
			":miniStaker,"+
			":assertionHash,"+
			":timeUnrivaled,"+
			":hasRival,"+
			":status,"+
			":hasLengthOneRival,"+
			":topLevelClaimHeight,"+
			":cumulativePathTimer"+
			")", convertEdgesToDBEdges(edges))
	if err != nil {
		log.Error("failed to insert into table", "err", err)
		return err
	}
	// Update the current table version,
	// before dropping the previous table version,
	// so that the deleted table version is not used
	d.versionMutex.Lock()
	d.currentTableVersion++
	d.versionMutex.Unlock()

	// Update the last update time, once the version has been updated
	d.lastUpdated = time.Now()
	// Drop the previous table version
	if currentTableVersion >= 0 {
		_, err = d.sqlDB.Query("DROP TABLE " + d.tableName + strconv.Itoa(currentTableVersion))
		if err != nil {
			log.Error("failed to drop table", "err", err)
			return err
		}
	}
	return nil
}

func (d *Database) close() {
	if err := d.sqlDB.Close(); err != nil {
		log.Error("failed to close database", "err", err)
	}
}

type DBEdge struct {
	ID                    []byte  `db:"id" json:"id"`
	Type                  string  `db:"type" json:"type"`
	StartCommitmentHash   []byte  `db:"startCommitmentHash" json:"startCommitmentHash"`
	StartCommitmentHeight int64   `db:"startCommitmentHeight" json:"startCommitmentHeight"`
	EndCommitmentHash     []byte  `db:"endCommitmentHash" json:"endCommitmentHash"`
	EndCommitmentHeight   int64   `db:"endCommitmentHeight" json:"endCommitmentHeight"`
	CreatedAtBlock        int64   `db:"createdAtBlock" json:"createdAtBlock"`
	MutualID              []byte  `db:"mutualId" json:"mutualId"`
	OriginID              []byte  `db:"originId" json:"originId"`
	ClaimID               []byte  `db:"claimId" json:"claimId"`
	HasChildren           bool    `db:"hasChildren" json:"hasChildren"`
	LowerChildID          []byte  `db:"lowerChildId" json:"lowerChildId"`
	UpperChildID          []byte  `db:"upperChildId" json:"upperChildId"`
	MiniStaker            []byte  `db:"miniStaker" json:"miniStaker"`
	AssertionHash         []byte  `db:"assertionHash" json:"assertionHash"`
	TimeUnrivaled         int64   `db:"timeUnrivaled" json:"timeUnrivaled"`
	HasRival              bool    `db:"hasRival" json:"hasRival"`
	Status                string  `db:"status" json:"status"`
	HasLengthOneRival     bool    `db:"hasLengthOneRival" json:"hasLengthOneRival"`
	TopLevelClaimHeight   []int64 `db:"topLevelClaimHeight" json:"topLevelClaimHeight"`
	CumulativePathTimer   int64   `db:"cumulativePathTimer" json:"cumulativePathTimer"`
}

type DBResponse struct {
	Edges       []DBEdge  `json:"result"`
	LastUpdated time.Time `json:"lastUpdated"`
}

func convertEdgeToDBEdge(edge *Edge) *DBEdge {
	return &DBEdge{
		ID:                    edge.ID.Bytes(),
		Type:                  edge.Type,
		StartCommitmentHash:   edge.StartCommitment.Hash.Bytes(),
		StartCommitmentHeight: int64(edge.StartCommitment.Height),
		EndCommitmentHash:     edge.EndCommitment.Hash.Bytes(),
		EndCommitmentHeight:   int64(edge.EndCommitment.Height),
		CreatedAtBlock:        int64(edge.CreatedAtBlock),
		MutualID:              edge.MutualID.Bytes(),
		OriginID:              edge.OriginID.Bytes(),
		ClaimID:               edge.ClaimID.Bytes(),
		HasChildren:           edge.HasChildren,
		LowerChildID:          edge.LowerChildID.Bytes(),
		UpperChildID:          edge.UpperChildID.Bytes(),
		MiniStaker:            edge.MiniStaker.Bytes(),
		AssertionHash:         edge.AssertionHash.Bytes(),
		TimeUnrivaled:         int64(edge.TimeUnrivaled),
		HasRival:              edge.HasRival,
		Status:                edge.Status,
		HasLengthOneRival:     edge.HasLengthOneRival,
		TopLevelClaimHeight:   heightSliceToInt64Slice(edge.TopLevelClaimHeight.ChallengeOriginHeights),
		CumulativePathTimer:   int64(edge.CumulativePathTimer),
	}
}

func convertEdgesToDBEdges(edges []*Edge) []*DBEdge {
	var result []*DBEdge
	for _, edge := range edges {
		result = append(result, convertEdgeToDBEdge(edge))
	}
	return result
}

func heightSliceToInt64Slice(i []protocol.Height) []int64 {
	var result []int64
	for _, v := range i {
		result = append(result, int64(v))
	}
	return result
}

// Query database with a sql query.
// Example query: "SELECT * FROM TABLE WHERE id='0x1234'"
// The table name is automatically replaced with the current table version, e.g. "TABLE" -> "edges0"
func (s *Server) queryDatabaseHandler(w http.ResponseWriter, r *http.Request) {
	s.database.versionMutex.RLock()
	defer s.database.versionMutex.RUnlock()
	query := mux.Vars(r)["query"]
	if query == "" {
		writeError(w, http.StatusBadRequest, errors.New("no query provided"))
		return
	}

	if s.database.currentTableVersion == -1 {
		writeError(w, http.StatusInternalServerError, errors.New("database not ready"))
		return
	}

	finalQuery := strings.Replace(query, "TABLE", s.database.tableName+strconv.Itoa(s.database.currentTableVersion), 1)
	var result []DBEdge
	err := s.database.sqlDB.Select(&result, finalQuery)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	err = writeJSONResponse(w, 200, DBResponse{
		Edges:       result,
		LastUpdated: s.database.lastUpdated,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
}

func (s *Server) updateDatabaseHandler(w http.ResponseWriter, r *http.Request) {
	// Only allow one update at a time
	if s.database.updateMutex.TryLock() {
		go func() {
			if err := s.database.Update(r.Context()); err != nil {
				log.Error("failed to update database", "err", err)
			}
		}()
	}
	w.WriteHeader(http.StatusOK)
}
