package redis_sentry

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/go-redis/redis/v8"
)

func TestTransaction(t *testing.T) {
	err := sentry.Init(sentry.ClientOptions{
		Debug:              true,
		Dsn:                "https://a5eac4fa3396cbfac8fb4baa6a9c03a3@o4504291071688704.ingest.sentry.io/4506715873804288",
		AttachStacktrace:   true,
		EnableTracing:      true,
		SampleRate:         1.0,
		TracesSampleRate:   1.0,
		ProfilesSampleRate: 1.0,
	})
	if err != nil {
		log.Fatalf("sentry.Init: %s", err)
	}
	defer sentry.Flush(2 * time.Second)

	ctx := context.Background()
	rdb := initRedis(ctx)
	testSimpleCmd(rdb)
	testCmdPipeline(rdb)
	sentry.CaptureMessage("test")
}

func testSimpleCmd(rdb *redis.Client) {
	ctx := context.Background()
	tx := sentry.StartTransaction(ctx, "test_signle_cmd")
	defer tx.Finish()

	ctx = tx.Context()

	status := rdb.Set(ctx, "a", 1, 2*time.Second)
	// assert ctx
	fmt.Printf("status %v\n", status)
}

func testCmdPipeline(rdb *redis.Client) {
	ctx := context.Background()
	tx := sentry.StartTransaction(ctx, "test_pipeline")
	defer tx.Finish()
	ctx = tx.Context()

	pipe := rdb.Pipeline()

	incr := pipe.Incr(ctx, "pipeline_counter")
	pipe.Expire(ctx, "pipeline_counter", time.Hour)

	cmds, err := pipe.Exec(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Printf("cmds %v\n", cmds)

	// The value is available only after Exec is called.
	fmt.Println(incr.Val())
}

func initRedis(ctx context.Context) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "",
		DB:       0,
	})

	pong, err := rdb.Ping(ctx).Result()
	if err != nil {
		fmt.Printf("connect to redis error")
	}
	fmt.Printf("connect to redis success, pong is %v\n", pong)

	hook := NewHook(rdb)

	rdb.AddHook(hook)

	return rdb

}
