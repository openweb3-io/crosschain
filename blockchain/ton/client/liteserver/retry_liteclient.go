package liteserver

import (
	"context"
	"fmt"
	"time"

	"github.com/xssnick/tonutils-go/tl"
	_ton "github.com/xssnick/tonutils-go/ton"
)

const (
	DefaultDelay = 100 * time.Millisecond
)

type RetryLiteClient struct {
	_ton.LiteClient

	MaxRetries int
	Timeout    time.Duration
}

var _ _ton.LiteClient = &RetryLiteClient{}

func (w *RetryLiteClient) QueryLiteserver(ctx context.Context, payload tl.Serializable, result tl.Serializable) error {
	var err error
	tries := 0
	ctx, cancel := context.WithTimeout(ctx, w.Timeout)
	defer cancel()

	backupCtx := ctx
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("query liteserver timeout: %w(%w)", ctx.Err(), err)
		default:
		}

		if tries > 0 {
			time.Sleep(DefaultDelay)
		}
		err = w.LiteClient.QueryLiteserver(ctx, payload, result)
		if w.MaxRetries > 0 && tries >= w.MaxRetries {
			return err
		}
		tries++

		if err != nil {
			return err
		}

		if w.MaxRetries == 0 {
			return nil
		}

		if resp, ok := result.(*tl.Serializable); ok {
			if res, ok := (*resp).(_ton.RunMethodResult); ok {
				switch res.ExitCode {
				case 0, 1:
					return nil

				// 	2 "stack underflow. Last op-code consume more elements than there are on stacks"
				// 	3 "stack overflow. More values have been stored on a stack than allowed by this version of TVM"
				// 	8 "cell overflow. Writing to builder is not possible since after operation there would be more than 1023 bits or 4 references"
				// 	9 "cell underflow. Read from slice primitive tried to read more bits or references than there are"
				case 2, 3, 8, 9:
					err = _ton.ContractExecError{
						Code: res.ExitCode,
					}

					// try with next best node
					var cerr error
					ctx, cerr = w.LiteClient.StickyContextNextNodeBalanced(ctx)
					if cerr != nil {
						ctx = backupCtx
					}

					continue
				default:
					return _ton.ContractExecError{
						Code: res.ExitCode,
					}
				}
			} else if res, ok := (*resp).(_ton.LSError); ok {
				switch res.Code {
				// code 502: backend node timeout
				// code 651: cannot load block: block is not in db (possibly out of sync)
				case 502, 651:
					err = res

					// try with next best node
					var cerr error
					ctx, cerr = w.LiteClient.StickyContextNextNodeBalanced(ctx)
					if cerr != nil {
						ctx = backupCtx
					}
					continue
				}

				return res
			}
		}

		return nil
	}
}
