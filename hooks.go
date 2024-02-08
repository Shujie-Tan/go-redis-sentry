package redis_sentry

import (
	"context"
	"strconv"
	"strings"

	"github.com/getsentry/sentry-go"
	"github.com/go-redis/redis/v8"
)

const (
	OP_DB_REDIS             = "db.redis"
	SPANDATA_DB_SYSTEM      = "db.system"
	SPANDATA_DB_NAME        = "db.name"
	SPANDATA_SERVER_ADDRESS = "server.address"
	SPANDATA_DB_OPERATION   = "db.operation"
)

var (
	_SINGLE_KEY_COMMANDS = []string{
		"decr", "decrby", "get", "incr", "incrby", "pttl", "set", "setex", "setnx", "ttl",
	}
	_MULTI_KEY_COMMANDS = []string{
		"del", "touch", "unlink",
	}
)

type redis_ctx_key struct{}

type sentryhook struct {
	rdb *redis.Client
}

// NewHook new hook for go-redis
//
// be care that the context should contain sentry transacton.Context info when calling the methods of redis.Client
func NewHook(rdb *redis.Client) redis.Hook {
	return sentryhook{rdb}
}

func (h sentryhook) BeforeProcess(ctx context.Context, cmd redis.Cmder) (context.Context, error) {
	tx := sentry.SpanFromContext(ctx)
	if tx != nil {
		span := tx.StartChild(OP_DB_REDIS, sentry.WithDescription(cmd.String()))
		h.setDBDataOnSpan(span)
		h.setClientData(span, cmd.Name(), cmd.Args())
		ctx = context.WithValue(ctx, redis_ctx_key{}, span)
	}
	return ctx, nil
}

func (h sentryhook) AfterProcess(ctx context.Context, cmd redis.Cmder) error {
	span, ok := ctx.Value(redis_ctx_key{}).(*sentry.Span)
	if ok {
		span.Finish()
	}
	return nil
}

func (h sentryhook) BeforeProcessPipeline(ctx context.Context, cmds []redis.Cmder) (context.Context, error) {
	tx := sentry.SpanFromContext(ctx)
	if tx != nil {
		description := "redis.pipeline.execute"
		span := tx.StartChild(OP_DB_REDIS, sentry.WithDescription(description))
		h.setDBDataOnSpan(span)
		h.setPipelineData(span, cmds)
		ctx = context.WithValue(ctx, redis_ctx_key{}, span)
	}
	return ctx, nil
}

func (h sentryhook) AfterProcessPipeline(ctx context.Context, cmds []redis.Cmder) error {
	span, ok := ctx.Value(redis_ctx_key{}).(*sentry.Span)
	if ok {
		span.Finish()
	}
	return nil
}

func (h sentryhook) setDBDataOnSpan(span *sentry.Span) {
	ops := h.rdb.Options()

	span.SetData(SPANDATA_DB_SYSTEM, "redis")
	span.SetData(SPANDATA_DB_NAME, strconv.Itoa(ops.DB))
	span.SetData(SPANDATA_SERVER_ADDRESS, ops.Addr)
}

func (h sentryhook) setClientData(span *sentry.Span, name string, args []interface{}) {
	if len(name) > 0 {
		span.SetTag("redis.command", name)
		span.SetTag(SPANDATA_DB_OPERATION, name)
	}

	if len(name) > 0 && len(args) > 1 {
		nameLow := strings.ToLower(name)
		if stringInSlice(nameLow, _SINGLE_KEY_COMMANDS) || stringInSlice(nameLow, _MULTI_KEY_COMMANDS) && len(args) == 2 {
			key, ok := args[1].(string)
			if ok {
				span.SetTag("redis.key", key)
			}
		}
	}
}

func (h sentryhook) setPipelineData(span *sentry.Span, cmds []redis.Cmder) {
	var commands string

	for _, cmd := range cmds {
		if len(commands) == 0 {
			commands = cmd.String()
		} else {
			// todo: should remove sensitive information
			commands += ", " + cmd.String()
		}
	}

	span.SetData("redis.commands", commands)
}
