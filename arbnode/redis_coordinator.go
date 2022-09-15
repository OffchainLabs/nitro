package arbnode

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"github.com/go-redis/redis/v8"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/redisutil"
)

const CHOSENSEQ_KEY string = "coordinator.chosen"              // Never overwritten. Expires or released only
const MSG_COUNT_KEY string = "coordinator.msgCount"            // Only written by sequencer holding CHOSEN key
const PRIORITIES_KEY string = "coordinator.priorities"         // Read only
const LIVELINESS_KEY_PREFIX string = "coordinator.liveliness." // Per server. Only written by self
const MESSAGE_KEY_PREFIX string = "coordinator.msg."           // Per Message. Only written by sequencer holding CHOSEN
const LIVELINESS_VAL string = "OK"
const INVALID_VAL string = "INVALID"
const INVALID_URL string = "<?INVALID-URL?>"

type RedisCoordinator struct {
	client redis.UniversalClient
}

func NewRedisCoordinator(redisUrl string) (*RedisCoordinator, error) {
	redisClient, err := redisutil.RedisClientFromURL(redisUrl)
	if err != nil {
		return nil, err
	}

	return &RedisCoordinator{
		client: redisClient,
	}, nil
}

func (c *RedisCoordinator) RecommendLiveSequencer(ctx context.Context) (string, error) {
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
	log.Info("no sequencer appears live on redis", "priorities", prioritiesString)
	return "", nil
}

func messageKeyFor(pos arbutil.MessageIndex) string {
	return fmt.Sprintf("%s%d", MESSAGE_KEY_PREFIX, pos)
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
	resBytes, err = c.signer.VerifyMessageSignature(nil, resBytes)
	if err != nil {
		return 0, err
	}
	if len(resBytes) != 8 {
		return 0, fmt.Errorf("unexpected msg count value length %v", len(resBytes))
	}
	return arbutil.MessageIndex(binary.BigEndian.Uint64(resBytes)), nil
}

func (c *SeqCoordinator) GetRemoteMsgCount() (arbutil.MessageIndex, error) {
	return c.getRemoteMsgCountImpl(c.GetContext(), c.client)
}
