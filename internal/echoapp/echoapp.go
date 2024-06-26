// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

// This pkg is a echo application that uses the Cartesi rollup HTTP API.
package echoapp

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	rollup "github.com/calindra/nonodo/internal/rollup"
)

// This worker uses the rollup API to implement an echo application.
// It uses the API rather than talking directly to the model so it can be used in integration tests.
type EchoAppWorker struct {
	RollupEndpoint string
	// Delay between requests 202 Accepted
	TimeoutDelay *time.Duration
	// Delay after find inspect requests
	TimeoutInspect *time.Duration
	// Delay after find advance requests
	TimeoutAdvance *time.Duration
}

func (w EchoAppWorker) String() string {
	return "echo"
}

func (w EchoAppWorker) delay(d *time.Duration) {
	if d != nil && *d > 0 {
		slog.Debug("echo: waiting for", slog.Duration("time", *d))
		time.Sleep(*d)
	}
}

func (w EchoAppWorker) Start(ctx context.Context, ready chan<- struct{}) error {
	slog.Debug("echo: starting echo application")
	client, err := rollup.NewClientWithResponses(w.RollupEndpoint)
	if err != nil {
		return fmt.Errorf("echo: %w", err)
	}

	ready <- struct{}{}

	finishReq := rollup.Finish{
		Status: rollup.Accept,
	}

	for {
		finishResp, err := client.FinishWithResponse(ctx, finishReq)
		if err != nil {
			return fmt.Errorf("echo: %w", err)
		}
		if finishResp.StatusCode() == http.StatusAccepted {
			slog.Debug("echo: waiting for next request")
			w.delay(w.TimeoutDelay)
			continue
		}
		if finishResp.StatusCode() != http.StatusOK {
			return fmt.Errorf("echo: invalid finish response: status=%v body=`%v`",
				finishResp.StatusCode(), string(finishResp.Body))
		}
		finishBody := finishResp.JSON200
		if finishBody == nil {
			return fmt.Errorf("echo: missing finish response body")
		}
		switch finishBody.RequestType {
		case rollup.AdvanceState:
			slog.Debug("echo: received advance request")
			advance, err := finishBody.Data.AsAdvance()
			if err != nil {
				return fmt.Errorf("echo: failed to parser advance: %w", err)
			}
			if err := handleAdvance(ctx, client, advance); err != nil {
				return err
			}
			w.delay(w.TimeoutAdvance)
		case rollup.InspectState:
			slog.Debug("echo: received inspect request")
			inspect, err := finishBody.Data.AsInspect()
			if err != nil {
				return fmt.Errorf("echo: failed to parser inspect: %w", err)
			}
			if err := handleInspect(ctx, client, inspect); err != nil {
				return err
			}
			w.delay(w.TimeoutInspect)
		default:
			return fmt.Errorf("echo: invalid request type: %v", finishBody.RequestType)
		}
	}
}

func handleAdvance(
	ctx context.Context,
	client *rollup.ClientWithResponses,
	advance rollup.Advance,
) error {
	slog.Info("echo: handling advance input")

	// add voucher
	voucherReq := rollup.Voucher{
		Destination: advance.Metadata.MsgSender,
		Payload:     advance.Payload,
	}
	voucherResp, err := client.AddVoucher(ctx, voucherReq)
	if err != nil {
		return fmt.Errorf("echo: %w", err)
	}
	if voucherResp.StatusCode != http.StatusOK {
		return fmt.Errorf("echo: failed to add report")
	}

	// add notice
	noticeReq := rollup.Notice{
		Payload: fmt.Sprintf("%sff", advance.Payload),
	}
	noticeResp, err := client.AddNotice(ctx, noticeReq)
	if err != nil {
		return fmt.Errorf("echo: %w", err)
	}
	if noticeResp.StatusCode != http.StatusOK {
		return fmt.Errorf("echo: failed to add notice")
	}

	// add report
	reportReq := rollup.Report{
		Payload: advance.Payload,
	}
	reportResp, err := client.AddReport(ctx, reportReq)
	if err != nil {
		return fmt.Errorf("echo: %w", err)
	}
	if reportResp.StatusCode != http.StatusOK {
		return fmt.Errorf("echo: failed to add report")
	}

	return nil
}

func handleInspect(
	ctx context.Context,
	client *rollup.ClientWithResponses,
	inspect rollup.Inspect,
) error {
	slog.Info("echo: handling inspect input")

	// add report
	reportReq := rollup.Report(inspect)
	reportResp, err := client.AddReport(ctx, reportReq)
	if err != nil {
		return fmt.Errorf("echo: %w", err)
	}
	if reportResp.StatusCode != http.StatusOK {
		return fmt.Errorf("echo: failed to add report")
	}

	return nil
}
