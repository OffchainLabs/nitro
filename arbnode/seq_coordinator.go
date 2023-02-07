// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/contracts"
	"github.com/offchainlabs/nitro/util/redisutil"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

var (
	isActiveSequencer = metrics.NewRegisteredGauge("arb/sequencer/active", nil)
)

type SeqCoordinator struct {
	stopwaiter.StopWaiter

	redisutil.RedisCoordinator

	sync             *SyncMonitor
	streamer         *TransactionStreamer
	sequencer        *Sequencer
	delayedSequencer *DelayedSequencer
	signer           *signature.SignVerify
	config           SeqCoordinatorConfig

	prevChosenSequencer string
	reportedAlive       bool

	lockoutUntil int64 // atomic

	chosenUpdateMutex sync.Mutex // manages access to chosenOneUpdate
	redisErrors       int        // error counter, from workthread
}

type SeqCoordinatorConfig struct {
	Enable                 bool                       `koanf:"enable"`
	ChosenHealthcheckAddr  string                     `koanf:"chosen-healthcheck-addr"`
	RedisUrl               string                     `koanf:"redis-url"`
	LockoutDuration        time.Duration              `koanf:"lockout-duration"`
	LockoutSpare           time.Duration              `koanf:"lockout-spare"`
	SeqNumDuration         time.Duration              `koanf:"seq-num-duration"`
	UpdateInterval         time.Duration              `koanf:"update-interval"`
	RetryInterval          time.Duration              `koanf:"retry-interval"`
	ShutdownHandoffTimeout time.Duration              `koanf:"shutdown-handoff-timeout"`
	SafeShutdownDelay      time.Duration              `koanf:"safe-shutdown-delay"`
	MaxMsgPerPoll          arbutil.MessageIndex       `koanf:"msg-per-poll"`
	MyUrlImpl              string                     `koanf:"my-url"`
	Signing                signature.SignVerifyConfig `koanf:"signer"`
}

func (c *SeqCoordinatorConfig) MyUrl() string {
	if c.MyUrlImpl == "" {
		return redisutil.INVALID_URL
	}

	return c.MyUrlImpl
}

func SeqCoordinatorConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultSeqCoordinatorConfig.Enable, "enable sequence coordinator")
	f.String(prefix+".redis-url", DefaultSeqCoordinatorConfig.RedisUrl, "the Redis URL to coordinate via")
	f.String(prefix+".chosen-healthcheck-addr", DefaultSeqCoordinatorConfig.ChosenHealthcheckAddr, "if non-empty, launch an HTTP service binding to this address that returns status code 200 when chosen and 503 otherwise")
	f.Duration(prefix+".lockout-duration", DefaultSeqCoordinatorConfig.LockoutDuration, "")
	f.Duration(prefix+".lockout-spare", DefaultSeqCoordinatorConfig.LockoutSpare, "")
	f.Duration(prefix+".seq-num-duration", DefaultSeqCoordinatorConfig.SeqNumDuration, "")
	f.Duration(prefix+".update-interval", DefaultSeqCoordinatorConfig.UpdateInterval, "")
	f.Duration(prefix+".retry-interval", DefaultSeqCoordinatorConfig.RetryInterval, "")
	f.Duration(prefix+".shutdown-handoff-timeout", DefaultSeqCoordinatorConfig.ShutdownHandoffTimeout, "the maximum amount of time to spend waiting for another sequencer to accept the lockout on shutdown")
	f.Duration(prefix+".safe-shutdown-delay", DefaultSeqCoordinatorConfig.SafeShutdownDelay, "if non-zero will add delay after transferring control")
	f.Uint64(prefix+".msg-per-poll", uint64(DefaultSeqCoordinatorConfig.MaxMsgPerPoll), "will only be marked live if not too far behind")
	f.String(prefix+".my-url", DefaultSeqCoordinatorConfig.MyUrlImpl, "url for this sequencer if it is the chosen")
	signature.SignVerifyConfigAddOptions(prefix+".signer", f)
}

var DefaultSeqCoordinatorConfig = SeqCoordinatorConfig{
	Enable:                 false,
	ChosenHealthcheckAddr:  "",
	RedisUrl:               "",
	LockoutDuration:        time.Minute,
	LockoutSpare:           30 * time.Second,
	SeqNumDuration:         24 * time.Hour,
	UpdateInterval:         250 * time.Millisecond,
	ShutdownHandoffTimeout: 30 * time.Second,
	SafeShutdownDelay:      5 * time.Second,
	RetryInterval:          time.Second,
	MaxMsgPerPoll:          2000,
	MyUrlImpl:              redisutil.INVALID_URL,
	Signing:                signature.DefaultSignVerifyConfig,
}

var TestSeqCoordinatorConfig = SeqCoordinatorConfig{
	Enable:                 false,
	RedisUrl:               redisutil.DefaultTestRedisURL,
	LockoutDuration:        time.Second * 2,
	LockoutSpare:           time.Millisecond * 10,
	SeqNumDuration:         time.Minute * 10,
	UpdateInterval:         time.Millisecond * 10,
	ShutdownHandoffTimeout: time.Duration(0),
	SafeShutdownDelay:      time.Duration(0),
	RetryInterval:          time.Millisecond * 3,
	MaxMsgPerPoll:          20,
	MyUrlImpl:              redisutil.INVALID_URL,
	Signing:                signature.DefaultSignVerifyConfig,
}

func NewSeqCoordinator(dataSigner signature.DataSignerFunc, bpvalidator *contracts.BatchPosterVerifier, streamer *TransactionStreamer, sequencer *Sequencer, sync *SyncMonitor, config SeqCoordinatorConfig) (*SeqCoordinator, error) {
	redisCoordinator, err := redisutil.NewRedisCoordinator(config.RedisUrl)
	if err != nil {
		return nil, err
	}
	signer, err := signature.NewSignVerify(&config.Signing, dataSigner, bpvalidator)
	if err != nil {
		return nil, err
	}
	coordinator := &SeqCoordinator{
		RedisCoordinator: *redisCoordinator,
		sync:             sync,
		streamer:         streamer,
		sequencer:        sequencer,
		config:           config,
		signer:           signer,
	}
	if sequencer != nil {
		sequencer.Pause()
	}
	streamer.SetSeqCoordinator(coordinator)
	return coordinator, nil
}

func (c *SeqCoordinator) SetDelayedSequencer(delayedSequencer *DelayedSequencer) {
	if c.Started() {
		panic("trying to set delayed sequencer after start")
	}
	if c.delayedSequencer != nil {
		panic("trying to set delayed sequencer when already set")
	}
	c.delayedSequencer = delayedSequencer
}

func StandaloneSeqCoordinatorInvalidateMsgIndex(ctx context.Context, redisClient redis.UniversalClient, keyConfig string, msgIndex arbutil.MessageIndex) error {
	signerConfig := signature.EmptySimpleHmacConfig
	if keyConfig == "" {
		signerConfig.Dangerous.DisableSignatureVerification = true
	} else {
		signerConfig.SigningKey = keyConfig
	}
	signer, err := signature.NewSimpleHmac(&signerConfig)
	if err != nil {
		return err
	}
	var msgIndexBytes [8]byte
	binary.BigEndian.PutUint64(msgIndexBytes[:], uint64(msgIndex))
	msg := []byte(redisutil.INVALID_VAL)
	sig, err := signer.SignMessage(msgIndexBytes[:], msg)
	if err != nil {
		return err
	}
	redisClient.Set(ctx, redisutil.MessageKeyFor(msgIndex), msg, DefaultSeqCoordinatorConfig.SeqNumDuration)
	redisClient.Set(ctx, redisutil.MessageSigKeyFor(msgIndex), sig, DefaultSeqCoordinatorConfig.SeqNumDuration)
	return nil
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

func (c *SeqCoordinator) msgCountToSignedBytes(msgCount arbutil.MessageIndex) ([]byte, error) {
	var msgCountBytes [8]byte
	binary.BigEndian.PutUint64(msgCountBytes[:], uint64(msgCount))
	sig, err := c.signer.SignMessage(msgCountBytes[:])
	if err != nil {
		return nil, err
	}
	return append(sig, msgCountBytes[:]...), nil
}

func (c *SeqCoordinator) signedBytesToMsgCount(ctx context.Context, data []byte) (arbutil.MessageIndex, error) {
	datalen := len(data)
	if datalen < 8 {
		return 0, errors.New("msgcount value too short")
	}
	msgCountBytes := data[datalen-8:]
	sig := data[:datalen-8]
	err := c.signer.VerifySignature(ctx, sig, msgCountBytes)
	if err != nil {
		return 0, err
	}
	return arbutil.MessageIndex(binary.BigEndian.Uint64(msgCountBytes)), nil
}

func (c *SeqCoordinator) chosenOneUpdate(ctx context.Context, msgCountExpected, msgCountToWrite arbutil.MessageIndex, lastmsg *arbstate.MessageWithMetadata) error {
	var messageData *string
	var messageSigData *string
	if lastmsg != nil {
		msgBytes, err := json.Marshal(lastmsg)
		if err != nil {
			return err
		}
		msgSig, err := c.signer.SignMessage(arbmath.UintToBytes(uint64(msgCountToWrite-1)), msgBytes)
		if err != nil {
			return err
		}
		if c.config.Signing.SymmetricSign {
			messageString := string(append(msgSig, msgBytes...))
			messageData = &messageString
		} else {
			messageString := string(msgBytes)
			sigString := string(msgSig)
			messageData = &messageString
			messageSigData = &sigString
		}
	}
	msgCountMsg, err := c.msgCountToSignedBytes(msgCountToWrite)
	if err != nil {
		return err
	}
	c.chosenUpdateMutex.Lock()
	defer c.chosenUpdateMutex.Unlock()
	lockoutUntil := time.Now().Add(c.config.LockoutDuration)
	err = c.Client.Watch(ctx, func(tx *redis.Tx) error {
		current, err := tx.Get(ctx, redisutil.CHOSENSEQ_KEY).Result()
		var wasEmpty bool
		if errors.Is(err, redis.Nil) {
			wasEmpty = true
			err = nil
		}
		if err != nil {
			return err
		}
		if !wasEmpty && (current != c.config.MyUrl()) {
			return fmt.Errorf("%w: failed to catch lock. redis shows chosen: %s", ErrRetrySequencer, current)
		}
		remoteMsgCount, err := c.getRemoteMsgCountImpl(ctx, tx)
		if err != nil {
			return err
		}
		if remoteMsgCount > msgCountExpected {
			if messageData == nil && c.CurrentlyChosen() {
				// this was called from update(), while msgCount was changed by a call from SequencingMessage
				// no need to do anything
				return nil
			}
			log.Info("coordinator failed to become main", "expected", msgCountExpected, "found", remoteMsgCount, "message is nil?", messageData == nil)
			return fmt.Errorf("%w: failed to catch lock. expected msg %d found %d", ErrRetrySequencer, msgCountExpected, remoteMsgCount)
		}
		pipe := tx.TxPipeline()
		initialDuration := c.config.LockoutDuration
		if initialDuration < 2*time.Second {
			initialDuration = 2 * time.Second
		}
		if wasEmpty {
			pipe.Set(ctx, redisutil.CHOSENSEQ_KEY, c.config.MyUrl(), initialDuration)
		}
		pipe.Set(ctx, redisutil.MSG_COUNT_KEY, msgCountMsg, c.config.SeqNumDuration)
		myLivelinessKey := redisutil.LivelinessKeyFor(c.config.MyUrl())
		pipe.Set(ctx, myLivelinessKey, redisutil.LIVELINESS_VAL, initialDuration)
		if messageData != nil {
			pipe.Set(ctx, redisutil.MessageKeyFor(msgCountToWrite-1), *messageData, c.config.SeqNumDuration)
			if messageSigData != nil {
				pipe.Set(ctx, redisutil.MessageSigKeyFor(msgCountToWrite-1), *messageSigData, c.config.SeqNumDuration)
			}
		}
		pipe.PExpireAt(ctx, redisutil.CHOSENSEQ_KEY, lockoutUntil)
		pipe.PExpireAt(ctx, myLivelinessKey, lockoutUntil)
		err = execTestPipe(pipe, ctx)
		if errors.Is(err, redis.TxFailedErr) {
			return fmt.Errorf("%w: failed to catch sequencer lock", ErrRetrySequencer)
		}
		if err != nil {
			return fmt.Errorf("chosen sequencer failed to update redis: %w", err)
		}
		return nil
	}, redisutil.CHOSENSEQ_KEY, redisutil.MSG_COUNT_KEY)

	if err != nil {
		return err
	}
	isActiveSequencer.Update(1)
	atomicTimeWrite(&c.lockoutUntil, lockoutUntil.Add(-c.config.LockoutSpare))
	return nil
}

func (c *SeqCoordinator) getRemoteMsgCountImpl(ctx context.Context, r redis.Cmdable) (arbutil.MessageIndex, error) {
	resStr, err := r.Get(ctx, redisutil.MSG_COUNT_KEY).Result()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return c.signedBytesToMsgCount(ctx, []byte(resStr))
}

func (c *SeqCoordinator) GetRemoteMsgCount() (arbutil.MessageIndex, error) {
	return c.getRemoteMsgCountImpl(c.GetContext(), c.Client)
}

func (c *SeqCoordinator) livelinessUpdate(ctx context.Context) error {
	myLivelinessKey := redisutil.LivelinessKeyFor(c.config.MyUrl())
	aliveUntil := time.Now().Add(c.config.LockoutDuration)
	pipe := c.Client.TxPipeline()
	initialDuration := c.config.LockoutDuration
	if initialDuration < 2*time.Second {
		initialDuration = 2 * time.Second
	}
	pipe.Set(ctx, myLivelinessKey, redisutil.LIVELINESS_VAL, initialDuration)
	pipe.PExpireAt(ctx, myLivelinessKey, aliveUntil)
	err := execTestPipe(pipe, ctx)
	if err != nil {
		return fmt.Errorf("liveliness failed to update redis: %w", err)
	}
	return nil
}

func (c *SeqCoordinator) chosenOneRelease(ctx context.Context) error {
	atomicTimeWrite(&c.lockoutUntil, time.Time{})
	isActiveSequencer.Update(0)
	releaseErr := c.Client.Watch(ctx, func(tx *redis.Tx) error {
		current, err := tx.Get(ctx, redisutil.CHOSENSEQ_KEY).Result()
		if errors.Is(err, redis.Nil) {
			return nil
		}
		if err != nil {
			return err
		}
		if current != c.config.MyUrl() {
			return nil
		}
		pipe := tx.TxPipeline()
		pipe.Del(ctx, redisutil.CHOSENSEQ_KEY)
		err = execTestPipe(pipe, ctx)
		if err != nil {
			return fmt.Errorf("chosen sequencer failed to update redis: %w", err)
		}
		return nil
	}, redisutil.CHOSENSEQ_KEY)
	if releaseErr == nil {
		return nil
	}
	// got error - was it still released?
	current, readErr := c.Client.Get(ctx, redisutil.CHOSENSEQ_KEY).Result()
	if errors.Is(readErr, redis.Nil) {
		return nil
	}
	if current != c.config.MyUrl() {
		return nil
	}
	return releaseErr
}

func (c *SeqCoordinator) livelinessRelease(ctx context.Context) error {
	myLivelinessKey := redisutil.LivelinessKeyFor(c.config.MyUrl())
	releaseErr := c.Client.Del(ctx, myLivelinessKey).Err()
	if releaseErr == nil {
		return nil
	}
	// got error - was it still deleted?
	readErr := c.Client.Get(ctx, myLivelinessKey).Err()
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
	if nextChosen != c.config.MyUrl() {
		// was the active sequencer, but no longer
		setPrevChosenTo := nextChosen
		if c.sequencer != nil {
			err := c.sequencer.ForwardTo(nextChosen)
			if err != nil {
				// The error was already logged in ForwardTo, just clean up state.
				// Setting prevChosenSequencer to an empty string will cause the next update to attempt to reconnect.
				setPrevChosenTo = ""
			}
		}
		if err := c.chosenOneRelease(ctx); err != nil {
			log.Warn("coordinator failed chosen one release", "err", err)
			return c.retryAfterRedisError()
		}
		c.prevChosenSequencer = setPrevChosenTo
		log.Info("released chosen-coordinator lock", "nextChosen", nextChosen)
		return c.noRedisError()
	}
	// Was, and still, the active sequencer
	if time.Now().Add(c.config.UpdateInterval / 3).After(atomicTimeRead(&c.lockoutUntil)) {
		// if we recently sequenced - no need for an update
		return c.noRedisError()
	}
	localMsgCount, err := c.streamer.GetMessageCount()
	if err != nil {
		log.Error("coordinator cannot read message count", "err", err)
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
	chosenSeq, err := c.RecommendLiveSequencer(ctx)
	if err != nil {
		log.Warn("coordinator failed finding live sequencer", "err", err)
		return c.retryAfterRedisError()
	}
	if c.prevChosenSequencer == c.config.MyUrl() {
		return c.updatePrevKnownChosen(ctx, chosenSeq)
	}
	if chosenSeq != c.config.MyUrl() && chosenSeq != c.prevChosenSequencer {
		var err error
		if c.sequencer != nil {
			err = c.sequencer.ForwardTo(chosenSeq)
		}
		if err == nil {
			c.prevChosenSequencer = chosenSeq
			log.Info("chosen sequencer changed", "chosen", chosenSeq)
		} else {
			// The error was already logged in ForwardTo, just clean up state.
			// Next run this will attempt to reconnect.
			c.prevChosenSequencer = ""
		}
	}

	// read messages from redis
	localMsgCount, err := c.streamer.GetMessageCount()
	if err != nil {
		log.Error("cannot read message count", "err", err)
		return c.config.UpdateInterval
	}
	remoteMsgCount, err := c.GetRemoteMsgCount()
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
		resString, msgReadErr = c.Client.Get(ctx, redisutil.MessageKeyFor(msgToRead)).Result()
		if msgReadErr != nil {
			log.Warn("coordinator failed reading message", "pos", msgToRead, "err", msgReadErr)
			break
		}
		rsBytes := []byte(resString)
		var sigString string
		var sigBytes []byte
		sigSeparateKey := true
		sigString, msgReadErr = c.Client.Get(ctx, redisutil.MessageSigKeyFor(msgToRead)).Result()
		if errors.Is(msgReadErr, redis.Nil) {
			// no separate signature. Try reading old-style sig
			if len(rsBytes) < 32 {
				log.Warn("signature not found for msg", "pos", msgToRead)
				msgReadErr = errors.New("signature not found")
				break
			}
			sigBytes = rsBytes[:32]
			rsBytes = rsBytes[32:]
			sigSeparateKey = false
		} else if msgReadErr != nil {
			log.Warn("coordinator failed reading sig", "pos", msgToRead, "err", msgReadErr)
			break
		} else {
			sigBytes = []byte(sigString)
		}
		msgReadErr = c.signer.VerifySignature(ctx, sigBytes, arbmath.UintToBytes(uint64(msgToRead)), rsBytes)
		if msgReadErr != nil {
			log.Warn("coordinator failed verifying message signature", "pos", msgToRead, "err", msgReadErr, "separate-key", sigSeparateKey)
			break
		}
		var message arbstate.MessageWithMetadata
		err = json.Unmarshal(rsBytes, &message)
		if err != nil {
			log.Warn("coordinator failed to parse message from redis", "pos", msgToRead, "err", err)
			msgReadErr = fmt.Errorf("failed to parse message: %w", err)
			// redis messages spelled "INVALID" will be parsed as invalid L1 message, but only one at a time
			if len(messages) > 0 || string(rsBytes) != redisutil.INVALID_VAL {
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

	if c.config.MyUrl() == redisutil.INVALID_URL {
		return c.noRedisError()
	}

	// can take over as main sequencer?
	if localMsgCount >= remoteMsgCount && chosenSeq == c.config.MyUrl() {
		if c.sequencer == nil {
			log.Error("myurl main sequencer, but no sequencer exists")
			return c.noRedisError()
		}
		// we're here because we don't currently hold the lock
		// sequencer is already either paused or forwarding
		c.sequencer.Pause()
		err := c.chosenOneUpdate(ctx, localMsgCount, localMsgCount, nil)
		if err != nil {
			// this could be just new messages we didn't get yet - even then, we should retry soon
			log.Info("sequencer failed to become chosen", "err", err, "msgcount", localMsgCount)
			// make sure we're marked alive
			if err := c.livelinessUpdate(ctx); err != nil {
				log.Warn("failed to update liveliness", "err", err)
			}
			c.prevChosenSequencer = ""
			return c.retryAfterRedisError()
		}
		log.Info("caught chosen-coordinator lock")
		if c.delayedSequencer != nil {
			err = c.delayedSequencer.ForceSequenceDelayed(ctx)
			if err != nil {
				log.Warn("failed sequencing delayed messages after catching lock", "err", err)
			}
		}
		c.sequencer.Activate()
		c.prevChosenSequencer = c.config.MyUrl()
		return c.noRedisError()
	}

	// update liveliness
	var livelinessErr error
	if c.sync.Synced() {
		livelinessErr = c.livelinessUpdate(ctx)
		if livelinessErr == nil {
			c.reportedAlive = true
		}
	} else if c.reportedAlive {
		livelinessErr = c.livelinessRelease(ctx)
		if livelinessErr == nil {
			c.reportedAlive = false
		}
	}
	if livelinessErr != nil {
		log.Warn("coordinator failed to post liveness", "err", livelinessErr)
	}

	if (livelinessErr != nil) || (msgReadErr != nil) {
		return c.retryAfterRedisError()
	}
	return c.noRedisError()
}

func (c *SeqCoordinator) DebugPrint() string {
	return fmt.Sprint("Url:", c.config.MyUrl(),
		" prevChosenSequencer:", c.prevChosenSequencer,
		" reportedAlive:", c.reportedAlive,
		" lockoutUntil:", c.lockoutUntil,
		" redisErrors:", c.redisErrors)
}

type seqCoordinatorChosenHealthcheck struct {
	c *SeqCoordinator
}

func (h seqCoordinatorChosenHealthcheck) ServeHTTP(response http.ResponseWriter, _ *http.Request) {
	if h.c.CurrentlyChosen() {
		response.WriteHeader(http.StatusOK)
	} else {
		response.WriteHeader(http.StatusServiceUnavailable)
	}
}

func (c *SeqCoordinator) launchHealthcheckServer(ctx context.Context) {
	server := &http.Server{
		Addr:              c.config.ChosenHealthcheckAddr,
		Handler:           seqCoordinatorChosenHealthcheck{c},
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		err := server.Shutdown(ctx)
		if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			log.Warn("error shutting down coordinator chosen healthcheck server", "err", err)
		}
	}()

	err := server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Warn("error serving coordinator chosen healthcheck server", "err", err)
	}
}

func (c *SeqCoordinator) Start(ctxIn context.Context) {
	c.StopWaiter.Start(ctxIn, c)
	c.CallIteratively(c.update)
	if c.config.ChosenHealthcheckAddr != "" {
		c.StopWaiter.LaunchThread(c.launchHealthcheckServer)
	}
}

// Calls check() every c.config.RetryInterval until it returns true, or the context times out.
func (c *SeqCoordinator) waitFor(ctx context.Context, check func() bool) {
	for {
		result := check()
		if result {
			return
		}
		select {
		case <-ctx.Done():
			// The caller should've already logged an info line with context about what it's waiting on
			log.Warn("timed out waiting")
			return
		case <-time.After(c.config.RetryInterval):
		}
	}
}

func (c *SeqCoordinator) PrepareForShutdown() {
	// normal context will be closed, use parent context
	parentCtx := c.StopWaiter.GetParentContext()
	handoffCtx, cancel := context.WithTimeout(c.StopWaiter.GetContext(), c.config.ShutdownHandoffTimeout)
	defer cancel()
	if c.CurrentlyChosen() && c.config.ShutdownHandoffTimeout != time.Duration(0) {
		log.Info("Waiting for an alternative sequencer in the priorities to acquire liveliness...", "timeout", c.config.ShutdownHandoffTimeout)
		c.waitFor(handoffCtx, func() bool {
			otherSeq, err := c.RecommendLiveSequencerIgnoring(handoffCtx, c.config.MyUrl())
			if err != nil {
				log.Warn("failed to find alternative sequencer", "err", err)
				return false
			}
			log.Info("found an alternative sequencer", "url", otherSeq)
			return true
		})
	}
	wasChosen := c.CurrentlyChosen()
	c.StopWaiter.StopAndWait()
	if c.CurrentlyChosen() {
		wasChosen = true
	}
	if c.reportedAlive {
		err := c.livelinessRelease(parentCtx)
		if err != nil {
			log.Warn("liveliness release failed", "err", err)
		}
	}
	if wasChosen {
		err := c.chosenOneRelease(parentCtx)
		if err != nil {
			log.Warn("chosen release failed", "err", err)
		}
		if c.config.ShutdownHandoffTimeout != time.Duration(0) {
			log.Info("Waiting for someone else to become the chosen sequencer...", "timeout", c.config.ShutdownHandoffTimeout)
			var newTarget string
			c.waitFor(handoffCtx, func() bool {
				chosen, err := c.CurrentChosenSequencer(handoffCtx)
				if err != nil {
					log.Warn("failed to get chosen sequencer", "err", err)
					return false
				}
				if chosen != "" && chosen != c.config.MyUrl() {
					log.Info("got new chosen sequencer", "url", chosen)
					newTarget = chosen
					return true
				}
				return false
			})
			if newTarget != "" {
				err := c.sequencer.ForwardTo(newTarget)
				if err != nil {
					log.Warn("setting forward address failed", "err", err)
				} else {
					log.Info("Waiting some more", "delay", c.config.SafeShutdownDelay, "nextChosen", newTarget)
					<-time.After(c.config.SafeShutdownDelay)
				}
			}
		}
	}
}

func (c *SeqCoordinator) StopAndWait() {
	if !c.StopWaiter.Stopped() {
		c.PrepareForShutdown()
	}
	_ = c.Client.Close()
}

func (c *SeqCoordinator) CurrentlyChosen() bool {
	return time.Now().Before(atomicTimeRead(&c.lockoutUntil))
}

func (c *SeqCoordinator) SequencingMessage(pos arbutil.MessageIndex, msg *arbstate.MessageWithMetadata) error {
	if !c.CurrentlyChosen() {
		return fmt.Errorf("%w: not main sequencer", ErrRetrySequencer)
	}
	if err := c.chosenOneUpdate(c.GetContext(), pos, pos+1, msg); err != nil {
		return err
	}
	return nil
}
