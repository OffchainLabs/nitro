package arbnode

import (
	"context"
	"crypto/subtle"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util"
)

const CHOSENSEQ_KEY string = "coordinator.chosen"              // Never overwritten. Expires or released only
const MSG_COUNT_KEY string = "coordinator.msgCount"            // Only written by sequencer holding CHOSEN key
const PRIORITIES_KEY string = "coordinator.priorities"         // Read only
const LIVELINESS_KEY_PREFIX string = "coordinator.liveliness." // Per server. Only written by self
const MESSAGE_KEY_PREFIX string = "coordinator.msg."           // Per Message. Only written by sequencer holding CHOSEN
const LIVELINESS_VAL string = "OK"
const INVALID_VAL string = "INVALID"

type SeqCoordinator struct {
	util.StopWaiter

	streamer                *TransactionStreamer
	sequencer               *Sequencer
	client                  redis.UniversalClient
	config                  SeqCoordinatorConfig
	signingKey              *[32]byte // if not nil, the redis message signing key
	fallbackVerificationKey *[32]byte

	prevChosenSequencer string
	reportedAlive       bool

	lockoutUntil int64 // atomic

	chosenUpdateMutex sync.Mutex // mannages access to chosenOneUpdate
	redisErrors       int        // error counter, from wrokthread
}

type SeqCoordinatorConfig struct {
	Enable                  bool                          `koanf:"enable"`
	RedisUrl                string                        `koanf:"redis-url"`
	LockoutDuration         time.Duration                 `koanf:"lockout-duration"`
	LockoutSpare            time.Duration                 `koanf:"lockout-spare"`
	SeqNumDuration          time.Duration                 `koanf:"seq-num-duration"`
	UpdateInterval          time.Duration                 `koanf:"update-interval"`
	RetryInterval           time.Duration                 `koanf:"retry-interval"`
	AllowedMsgLag           arbutil.MessageIndex          `koanf:"allowed-msg-lag"`
	MaxMsgPerPoll           arbutil.MessageIndex          `koanf:"msg-per-poll"`
	MyUrl                   string                        `koanf:"my-url"`
	SigningKey              string                        `koanf:"signing-key"`
	FallbackVerificationKey string                        `koanf:"fallback-verification-key"`
	Dangerous               SeqCoordinatorDangerousConfig `koanf:"dangerous"`
}

type SeqCoordinatorDangerousConfig struct {
	DisableSignatureVerificaiton bool `koanf:"disable-signature-verification"`
}

func SeqCoordinatorConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultSeqCoordinatorConfig.Enable, "enable sequence coordinator")
	f.Duration(prefix+".lockout-duration", DefaultSeqCoordinatorConfig.LockoutDuration, "")
	f.Duration(prefix+".lockout-spare", DefaultSeqCoordinatorConfig.LockoutSpare, "")
	f.Duration(prefix+".seq-num-duration", DefaultSeqCoordinatorConfig.SeqNumDuration, "")
	f.Duration(prefix+".update-interval", DefaultSeqCoordinatorConfig.UpdateInterval, "")
	f.Duration(prefix+".retry-interval", DefaultSeqCoordinatorConfig.RetryInterval, "")
	f.Uint16(prefix+".allowed-msg-lag", uint16(DefaultSeqCoordinatorConfig.AllowedMsgLag), "will only be marked live if not too far behind")
	f.Uint16(prefix+".msg-per-poll", uint16(DefaultSeqCoordinatorConfig.MaxMsgPerPoll), "will only be marked live if not too far behind")
	f.String(prefix+".my-url", DefaultSeqCoordinatorConfig.MyUrl, "a 32-byte (64-character) hex string used to sign messages, or a path to a file containing it")
	f.String(prefix+".signing-key", DefaultSeqCoordinatorConfig.SigningKey, "")
	SeqCoordinatorDangerousConfigAddOptions(prefix+".dangerous", f)
}

func SeqCoordinatorDangerousConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".disable-signature-verification", DefaultSeqCoordinatorDangerousConfig.DisableSignatureVerificaiton, "disable message signature verification")
}

var DefaultSeqCoordinatorConfig = SeqCoordinatorConfig{
	Enable:          false,
	RedisUrl:        "",
	LockoutDuration: time.Duration(5) * time.Minute,
	LockoutSpare:    time.Duration(30) * time.Second,
	SeqNumDuration:  time.Duration(24) * time.Hour,
	UpdateInterval:  time.Duration(10) * time.Second,
	RetryInterval:   time.Second,
	AllowedMsgLag:   200,
	MaxMsgPerPoll:   120,
	MyUrl:           "",
	SigningKey:      "",
	Dangerous:       DefaultSeqCoordinatorDangerousConfig,
}

var DefaultSeqCoordinatorDangerousConfig = SeqCoordinatorDangerousConfig{
	DisableSignatureVerificaiton: false,
}

var TestSeqCoordinatorConfig = SeqCoordinatorConfig{
	Enable:          false,
	RedisUrl:        "redis://localhost:6379/0",
	LockoutDuration: time.Millisecond * 500,
	LockoutSpare:    time.Millisecond * 10,
	SeqNumDuration:  time.Minute * 10,
	UpdateInterval:  time.Millisecond * 10,
	RetryInterval:   time.Millisecond * 3,
	AllowedMsgLag:   5,
	MaxMsgPerPoll:   20,
	MyUrl:           "",
	SigningKey:      "b561f5d5d98debc783aa8a1472d67ec3bcd532a1c8d95e5cb23caa70c649f7c9",
	Dangerous: SeqCoordinatorDangerousConfig{
		DisableSignatureVerificaiton: false,
	},
}

var keyIsHexRegex = regexp.MustCompile("^[a-fA-F0-9]{64}$")

func loadSigningKey(keyConfig string) (*[32]byte, error) {
	if keyConfig == "" {
		return nil, nil
	}
	keyIsHex := keyIsHexRegex.Match([]byte(keyConfig))
	var keyString string
	if keyIsHex {
		keyString = keyConfig
	} else {
		contents, err := ioutil.ReadFile(keyConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to read signing key file: %w", err)
		}
		s := strings.TrimSpace(string(contents))
		if keyIsHexRegex.Match([]byte(s)) {
			keyString = s
		} else {
			return nil, errors.New("signing key file contents are not 32 bytes of hex")
		}
	}
	var b [32]byte = common.HexToHash(keyString)
	return &b, nil
}

func NewSeqCoordinator(streamer *TransactionStreamer, sequencer *Sequencer, config SeqCoordinatorConfig) (*SeqCoordinator, error) {
	redisOptions, err := redis.ParseURL(config.RedisUrl)
	if err != nil {
		return nil, err
	}
	signingKey, err := loadSigningKey(config.SigningKey)
	if err != nil {
		return nil, err
	}
	fallbackVerificationKey, err := loadSigningKey(config.FallbackVerificationKey)
	if err != nil {
		return nil, err
	}
	coordinator := &SeqCoordinator{
		streamer:                streamer,
		sequencer:               sequencer,
		client:                  redis.NewClient(redisOptions),
		config:                  config,
		signingKey:              signingKey,
		fallbackVerificationKey: fallbackVerificationKey,
	}
	streamer.SetSeqCoordinator(coordinator)
	return coordinator, nil
}

func StandaloneSeqCoordinatorInvalidateMsgIndex(ctx context.Context, redisUrl string, keyConfig string, msgIndex arbutil.MessageIndex) error {
	redisOptions, err := redis.ParseURL(redisUrl)
	if err != nil {
		return err
	}
	r := redis.NewClient(redisOptions)
	signingKey, err := loadSigningKey(keyConfig)
	if err != nil {
		return err
	}
	msg := []byte(INVALID_VAL)
	var hmac [32]byte
	if signingKey != nil {
		var msgIndexBytes [8]byte
		binary.BigEndian.PutUint64(msgIndexBytes[:], uint64(msgIndex))
		hmac = crypto.Keccak256Hash(signingKey[:], msgIndexBytes[:], msg)
	}
	data := append(hmac[:], msg...)
	r.Set(ctx, messageKeyFor(msgIndex), data, DefaultSeqCoordinatorConfig.SeqNumDuration)
	return nil
}

func (c *SeqCoordinator) recommendLiveSequencer(ctx context.Context) (string, error) {
	prioritiesString, err := c.client.Get(ctx, PRIORITIES_KEY).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			err = errors.New("sequencer priorities unset")
		}
		return "", err
	}
	priorities := strings.Split(prioritiesString, ",")
	for _, url := range priorities {
		err := c.client.Get(ctx, livelinessKeyFor(url)).Err()
		if errors.Is(err, redis.Nil) { // liveliness not set
			continue
		}
		if err != nil {
			return "", err
		}
		return url, nil
	}
	log.Info("no sequencer appears live on redis", "priorities", prioritiesString, "self", c.config.MyUrl)
	return "", nil
}

func atomicTimeWrite(addr *int64, t time.Time) {
	asint64 := t.UnixMilli()
	atomic.StoreInt64(addr, asint64)
}

// notice: It is possible for two consecutive reads to get decreasing values. That shouldn't matter.
func atomicTimeRead(addr *int64) time.Time {
	asint64 := atomic.LoadInt64(addr)
	return time.UnixMilli(asint64)
}

func livelinessKeyFor(url string) string { return LIVELINESS_KEY_PREFIX + url }

func messageKeyFor(pos arbutil.MessageIndex) string {
	return fmt.Sprintf("%s%d", MESSAGE_KEY_PREFIX, pos)
}

func execTestPipe(pipe redis.Pipeliner, ctx context.Context) error {
	cmders, err := pipe.Exec(ctx)
	if err != nil {
		return err
	}
	for _, cmder := range cmders {
		if err := cmder.Err(); err != nil {
			return err
		}
	}
	return nil
}

// On success, extracts the message from the message+signature data passed in, and returns it
func (c *SeqCoordinator) verifyMessageSignature(prefix []byte, data []byte) ([]byte, error) {
	if len(data) < 32 {
		return nil, errors.New("data is too short to contain message signature")
	}
	msg := data[32:]
	if c.config.Dangerous.DisableSignatureVerificaiton || c.signingKey == nil {
		return msg, nil
	}
	var haveHmac common.Hash
	copy(haveHmac[:], data[:32])

	expectHmac := crypto.Keccak256Hash(c.signingKey[:], prefix, msg)
	if subtle.ConstantTimeCompare(expectHmac[:], haveHmac[:]) == 1 {
		return msg, nil
	}

	if c.fallbackVerificationKey != nil {
		expectHmac = crypto.Keccak256Hash(c.fallbackVerificationKey[:], prefix, msg)
		if subtle.ConstantTimeCompare(expectHmac[:], haveHmac[:]) == 1 {
			return msg, nil
		}
	}

	if haveHmac == (common.Hash{}) {
		return nil, errors.New("no HMAC signature present but signature verification is enabled")
	} else {
		return nil, errors.New("HMAC signature doesn't match expected value(s)")
	}
}

func (c *SeqCoordinator) signMessage(prefix []byte, msg []byte) []byte {
	var hmac [32]byte
	if c.signingKey != nil {
		hmac = crypto.Keccak256Hash(c.signingKey[:], prefix, msg)
	}
	return append(hmac[:], msg...)
}

func (c *SeqCoordinator) chosenOneUpdate(ctx context.Context, msgCountExpected, msgCountToWrite arbutil.MessageIndex, lastmsg *arbstate.MessageWithMetadata) error {
	var messageData *string
	if lastmsg != nil {
		msgBytes, err := json.Marshal(lastmsg)
		if err != nil {
			return err
		}

		var msgCountBytes [8]byte
		binary.BigEndian.PutUint64(msgCountBytes[:], uint64(msgCountToWrite-1))
		msgBytes = c.signMessage(msgCountBytes[:], msgBytes)

		messageString := string(msgBytes)
		messageData = &messageString
	}
	c.chosenUpdateMutex.Lock()
	defer c.chosenUpdateMutex.Unlock()
	lockoutUntil := time.Now().Add(c.config.LockoutDuration)
	err := c.client.Watch(ctx, func(tx *redis.Tx) error {
		current, err := tx.Get(ctx, CHOSENSEQ_KEY).Result()
		var wasEmpty bool
		if errors.Is(err, redis.Nil) {
			wasEmpty = true
			err = nil
		}
		if err != nil {
			return err
		}
		if !wasEmpty && (current != c.config.MyUrl) {
			return fmt.Errorf("%w: redis shows chosen: %s", ErrNotMainSequencer, current)
		}
		remoteMsgCount, err := c.getRemoteMsgCountImpl(ctx, tx)
		if err != nil {
			return err
		}
		if remoteMsgCount > msgCountExpected {
			log.Info("coordinator failed to become main", "expected", msgCountExpected, "found", remoteMsgCount, "message is nil?", messageData == nil)
			return fmt.Errorf("%w: expected msg %d found %d", ErrNotMainSequencer, msgCountExpected, remoteMsgCount)
		}
		pipe := tx.TxPipeline()
		initialDuration := c.config.LockoutDuration
		if initialDuration < 2*time.Second {
			initialDuration = 2 * time.Second
		}
		if wasEmpty {
			pipe.Set(ctx, CHOSENSEQ_KEY, c.config.MyUrl, initialDuration)
		}
		var msgCountBytes [8]byte
		binary.BigEndian.PutUint64(msgCountBytes[:], uint64(msgCountToWrite))
		pipe.Set(ctx, MSG_COUNT_KEY, c.signMessage(nil, msgCountBytes[:]), c.config.SeqNumDuration)
		myLivelinessKey := livelinessKeyFor(c.config.MyUrl)
		pipe.Set(ctx, myLivelinessKey, LIVELINESS_VAL, initialDuration)
		if messageData != nil {
			pipe.Set(ctx, messageKeyFor(msgCountToWrite-1), *messageData, c.config.SeqNumDuration)
		}
		pipe.PExpireAt(ctx, CHOSENSEQ_KEY, lockoutUntil)
		pipe.PExpireAt(ctx, myLivelinessKey, lockoutUntil)
		err = execTestPipe(pipe, ctx)
		if errors.Is(err, redis.TxFailedErr) {
			return fmt.Errorf("%w: transaction failed", ErrNotMainSequencer)
		}
		if err != nil {
			return fmt.Errorf("chosen sequencer failed to update redis: %w", err)
		}
		return nil
	}, CHOSENSEQ_KEY, MSG_COUNT_KEY)

	if err != nil {
		return err
	}
	atomicTimeWrite(&c.lockoutUntil, lockoutUntil.Add(-c.config.LockoutSpare))
	return nil
}

func (c *SeqCoordinator) getRemoteMsgCountImpl(ctx context.Context, r redis.Cmdable) (arbutil.MessageIndex, error) {
	resStr, err := r.Get(ctx, MSG_COUNT_KEY).Result()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	resBytes := []byte(resStr)
	resBytes, err = c.verifyMessageSignature(nil, []byte(resBytes))
	if err != nil {
		return 0, err
	}
	if len(resBytes) != 8 {
		return 0, fmt.Errorf("unexpected msg count value length %v", len(resBytes))
	}
	return arbutil.MessageIndex(binary.BigEndian.Uint64(resBytes)), nil
}

func (c *SeqCoordinator) GetRemoteMsgCount(ctx context.Context) (arbutil.MessageIndex, error) {
	return c.getRemoteMsgCountImpl(ctx, c.client)
}

func (c *SeqCoordinator) livelinessUpdate(ctx context.Context) error {
	myLivelinessKey := livelinessKeyFor(c.config.MyUrl)
	aliveUntil := time.Now().Add(c.config.LockoutDuration)
	pipe := c.client.TxPipeline()
	initialDuration := c.config.LockoutDuration
	if initialDuration < 2*time.Second {
		initialDuration = 2 * time.Second
	}
	pipe.Set(ctx, myLivelinessKey, LIVELINESS_VAL, initialDuration)
	pipe.PExpireAt(ctx, myLivelinessKey, aliveUntil)
	err := execTestPipe(pipe, ctx)
	if err != nil {
		return fmt.Errorf("liveliness failed to update redis: %w", err)
	}
	return nil
}

func (c *SeqCoordinator) chosenOneRelease(ctx context.Context) error {
	releaseErr := c.client.Watch(ctx, func(tx *redis.Tx) error {
		current, err := tx.Get(ctx, CHOSENSEQ_KEY).Result()
		if errors.Is(err, redis.Nil) {
			return nil
		}
		if err != nil {
			return err
		}
		if current != c.config.MyUrl {
			return nil
		}
		pipe := tx.TxPipeline()
		pipe.Del(ctx, CHOSENSEQ_KEY)
		err = execTestPipe(pipe, ctx)
		if err != nil {
			return fmt.Errorf("chosen sequencer failed to update redis: %w", err)
		}
		return nil
	}, CHOSENSEQ_KEY)
	if releaseErr == nil {
		return nil
	}
	// got error - was it still released?
	current, readErr := c.client.Get(ctx, CHOSENSEQ_KEY).Result()
	if errors.Is(readErr, redis.Nil) {
		return nil
	}
	if current != c.config.MyUrl {
		return nil
	}
	return releaseErr
}

func (c *SeqCoordinator) livelinessRelease(ctx context.Context) error {
	myLivelinessKey := livelinessKeyFor(c.config.MyUrl)
	releaseErr := c.client.Del(ctx, myLivelinessKey).Err()
	if releaseErr == nil {
		return nil
	}
	// got error - was it still deleted?
	readErr := c.client.Get(ctx, myLivelinessKey).Err()
	if errors.Is(readErr, redis.Nil) {
		return nil
	}
	return releaseErr
}

func (c *SeqCoordinator) retryAfterRedisError() time.Duration {
	c.redisErrors++
	retryIn := c.config.RetryInterval * time.Duration(c.redisErrors)
	if retryIn > c.config.UpdateInterval {
		retryIn = c.config.UpdateInterval
	}
	return retryIn
}

func (c *SeqCoordinator) noRedisError() time.Duration {
	c.redisErrors = 0
	return c.config.UpdateInterval
}

// update for the prev known-chosen sequencer (no need to load new messages)
func (c *SeqCoordinator) updatePrevKnownChosen(ctx context.Context, nextChosen string) time.Duration {
	if nextChosen != c.config.MyUrl {
		// was the active sequencer, but no longer
		atomicTimeWrite(&c.lockoutUntil, time.Time{})
		c.sequencer.ForwardTo(nextChosen)
		if err := c.chosenOneRelease(ctx); err != nil {
			log.Warn("coordinator failed chosen one release", "err", err)
			return c.retryAfterRedisError()
		}
		c.prevChosenSequencer = nextChosen
		return c.noRedisError()
	}
	// Was, and still, the active sequencer
	if time.Now().Add(c.config.UpdateInterval / 3).After(atomicTimeRead(&c.lockoutUntil)) {
		// if we recently sequenced - no need for an update
		return c.noRedisError()
	}
	localMsgCount, err := c.streamer.GetMessageCount()
	if err != nil {
		log.Crit("coordinator cannot read message count", "err", err)
		return c.config.UpdateInterval
	}
	err = c.chosenOneUpdate(ctx, localMsgCount, localMsgCount, nil)
	if err != nil {
		log.Warn("coordinator failed chosen-one keepalive", "err", err)
		return c.retryAfterRedisError()
	}
	c.reportedAlive = true
	return c.noRedisError()
}

func (c *SeqCoordinator) update(ctx context.Context) time.Duration {
	chosenSeq, err := c.recommendLiveSequencer(ctx)
	if err != nil {
		log.Warn("coordinator failed finding live sequencer", "err", err)
		return c.retryAfterRedisError()
	}
	if c.prevChosenSequencer == c.config.MyUrl {
		return c.updatePrevKnownChosen(ctx, chosenSeq)
	}
	if chosenSeq != c.config.MyUrl && chosenSeq != c.prevChosenSequencer {
		c.sequencer.ForwardTo(chosenSeq)
		c.prevChosenSequencer = chosenSeq
	}

	// read messages from redis
	localMsgCount, err := c.streamer.GetMessageCount()
	if err != nil {
		log.Crit("cannot read message count", "err", err)
		return c.config.UpdateInterval
	}
	remoteMsgCount, err := c.GetRemoteMsgCount(ctx)
	if err != nil {
		log.Warn("cannot get remote message count", "err", err)
		return c.retryAfterRedisError()
	}
	readUntil := remoteMsgCount
	if readUntil > localMsgCount+c.config.MaxMsgPerPoll {
		readUntil = localMsgCount + c.config.MaxMsgPerPoll
	}
	var messages []arbstate.MessageWithMetadata
	msgToRead := localMsgCount
	var msgReadErr error
	for msgToRead < readUntil {
		var resString string
		resString, msgReadErr = c.client.Get(ctx, messageKeyFor(msgToRead)).Result()
		if msgReadErr != nil {
			log.Warn("coordinator failed reading message", "pos", msgToRead, "err", msgReadErr)
			break
		}
		rsBytes := []byte(resString)
		var msgToReadBytes [8]byte
		binary.BigEndian.PutUint64(msgToReadBytes[:], uint64(msgToRead))
		rsBytes, msgReadErr = c.verifyMessageSignature(msgToReadBytes[:], rsBytes)
		if msgReadErr != nil {
			log.Warn("coordinator failed verifying message signature", "pos", msgToRead, "err", msgReadErr)
			break
		}
		var message arbstate.MessageWithMetadata
		err = json.Unmarshal(rsBytes, &message)
		if err != nil {
			log.Warn("coordinator failed to parse message from redis", "pos", msgToRead, "err", err)
			msgReadErr = fmt.Errorf("failed to parse message: %w", err)
			// redis messages spelled "INVALID" will be parsed as invalid L1 message, but only one at a time
			if len(messages) > 0 || string(rsBytes) != INVALID_VAL {
				break
			}
			lastDelayedMsg := uint64(0)
			if msgToRead > 0 {
				prevMsg, err := c.streamer.GetMessage(msgToRead - 1)
				if err != nil {
					log.Error("coordinator failed to get msg", "pos", msgToRead-1)
					break
				}
				lastDelayedMsg = prevMsg.DelayedMessagesRead
			}
			message = arbstate.MessageWithMetadata{
				Message:             arbstate.InvalidL1Message,
				DelayedMessagesRead: lastDelayedMsg,
			}
		}
		messages = append(messages, message)
		msgToRead++
	}
	if len(messages) > 0 {
		if err := c.streamer.AddMessages(localMsgCount, false, messages); err != nil {
			log.Warn("coordinator failed to add messages", "err", err, "pos", localMsgCount, "length", len(messages))
		} else {
			localMsgCount = msgToRead
		}
	}

	// can take over as main sequencer?
	if localMsgCount >= remoteMsgCount && chosenSeq == c.config.MyUrl {
		err := c.chosenOneUpdate(ctx, localMsgCount, localMsgCount, nil)
		if err != nil {
			// this could be just new messages we didn't get yet - even then, we should retry soon
			log.Info("sequencer failed to become chosen", "err", err, "msgcount", localMsgCount)
			// make sure we're marked alive
			if err := c.livelinessUpdate(ctx); err != nil {
				log.Warn("failed to update liveliness", "err", err)
			}
			return c.retryAfterRedisError()
		}
		c.sequencer.DontForward()
		c.prevChosenSequencer = c.config.MyUrl
		return c.noRedisError()
	}

	// update liveliness
	var livelinessErr error
	if localMsgCount+c.config.AllowedMsgLag < remoteMsgCount {
		if c.reportedAlive {
			livelinessErr = c.livelinessRelease(ctx)
			if livelinessErr == nil {
				c.reportedAlive = false
			}
		}
	} else {
		livelinessErr = c.livelinessUpdate(ctx)
		if livelinessErr == nil {
			c.reportedAlive = true
		}
	}
	if livelinessErr != nil {
		log.Warn("coordinator failed to post liveness", "err", err)
	}

	if (livelinessErr != nil) || (msgReadErr != nil) {
		return c.retryAfterRedisError()
	}
	return c.noRedisError()
}

func (c *SeqCoordinator) DebugPrint() string {
	return fmt.Sprint("Url:", c.config.MyUrl,
		" prevChosenSequencer:", c.prevChosenSequencer,
		" reportedAlive:", c.reportedAlive,
		" lockoutUntil:", c.lockoutUntil,
		" redisErrors:", c.redisErrors)
}

func (c *SeqCoordinator) Start(ctxIn context.Context) {
	c.StopWaiter.Start(ctxIn)
	c.CallIteratively(c.update)
}

func (c *SeqCoordinator) StopAndWait() {
	c.StopWaiter.StopAndWait()
	c.client.Close()
}

var ErrNotMainSequencer = errors.New("not main sequencer")

func (c *SeqCoordinator) CurrentlyChosen() bool {
	if time.Now().Before(atomicTimeRead(&c.lockoutUntil)) {
		return true
	}
	return false
}

func (c *SeqCoordinator) SequencingMessage(pos arbutil.MessageIndex, msg *arbstate.MessageWithMetadata) error {
	if !c.CurrentlyChosen() {
		return ErrNotMainSequencer
	}
	if err := c.chosenOneUpdate(c.GetContext(), pos, pos+1, msg); err != nil {
		return err
	}
	return nil
}
