package reader

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	convenience "github.com/calindra/nonodo/internal/convenience/model"
	"github.com/calindra/nonodo/internal/convenience/services"
	"github.com/calindra/nonodo/internal/graphile"
	graphql "github.com/calindra/nonodo/internal/reader/model"
	"github.com/ethereum/go-ethereum/common"
)

type AdapterV2 struct {
	convenienceService *services.ConvenienceService
	graphileClient     graphile.GraphileClient
	InputBlobAdapter   InputBlobAdapter
}

type InputByIdResponse struct {
	Data struct {
		Inputs struct {
			Edges []struct {
				Node struct {
					Index  int    `json:"index"`
					Blob   string `json:"blob"`
					Status string `json:"status"`
				} `json:"node"`
			} `json:"edges"`
		} `json:"inputs"`
	} `json:"data"`
}

type ReportByIdResponse struct {
	Data struct {
		Reports struct {
			Edges []struct {
				Node struct {
					Index      int    `json:"index"`
					Blob       string `json:"blob"`
					InputIndex int    `json:"inputIndex"`
				} `json:"node"`
			} `json:"edges"`
		} `json:"reports"`
	} `json:"data"`
}

type ProofByIndexes struct {
	Data struct {
		Proof graphql.Proof
	}
}

func NewAdapterV2(
	convenienceService *services.ConvenienceService,
	graphileClient graphile.GraphileClient,
	inputBlobAdapter InputBlobAdapter,
) Adapter {
	slog.Debug("NewAdapterV2")
	return AdapterV2{
		convenienceService: convenienceService,
		graphileClient:     graphileClient,
		InputBlobAdapter:   inputBlobAdapter,
	}
}

// GetProof implements Adapter.
func (a AdapterV2) GetProof(ctx context.Context, inputIndex int, outputIndex int) (*graphql.Proof, error) {
	query := `
	query ProofQuery($inputIndex: Int!, $outputIndex: Int!) {
		proof(inputIndex: $inputIndex, outputIndex: $outputIndex) {
			nodeId
			inputIndex
			outputIndex
			firstInput
			lastInput
			validityInputIndexWithinEpoch
			validityOutputIndexWithinInput
			validityOutputHashesRootHash
			validityOutputEpochRootHash
			validityMachineStateHash
			validityOutputHashInOutputHashesSiblings
			validityOutputHashesInEpochSiblings
		}
	}`
	variables := map[string]interface{}{
		"inputIndex":  inputIndex,
		"outputIndex": outputIndex,
	}
	payload, err := json.Marshal(map[string]interface{}{
		"operationName": nil,
		"query":         query,
		"variables":     variables,
	})
	if err != nil {
		slog.Error("Error marshalling JSON:", "error", err)
		return nil, err
	}
	response, err := a.graphileClient.Post(payload)
	slog.Debug("Proof", "response", string(response))
	if err != nil {
		slog.Error("Error calling Graphile Reports", "error", err)
		return nil, err
	}
	var theProof ProofByIndexes
	err = json.Unmarshal(response, &theProof)

	if err != nil {
		slog.Error("Error decoding JSON:", "error", err)
		return nil, err
	}

	return &theProof.Data.Proof, nil
}

func (a AdapterV2) GetReport(reportIndex int, inputIndex int) (*graphql.Report, error) {
	requestBody := []byte(fmt.Sprintf(`
    {
		"query": "query { reports(condition: {index: %d, inputIndex: %d}) { edges { node { index blob inputIndex}}}}"
		
	}`, reportIndex, inputIndex))

	response, err := a.graphileClient.Post(requestBody)

	if err != nil {
		slog.Error("Error calling Graphile Reports", "error", err)
		return nil, err
	}

	var reportByIdResponse ReportByIdResponse
	err = json.Unmarshal(response, &reportByIdResponse)

	if err != nil {
		slog.Error("Error decoding JSON:", "error", err)
		return nil, err
	}

	if len(reportByIdResponse.Data.Reports.Edges) > 0 {
		return convertReport(reportByIdResponse.Data.Reports.Edges[0].Node)
	}
	return nil, nil
}

func (a AdapterV2) GetReports(
	ctx context.Context,
	first *int,
	last *int,
	after *string,
	before *string,
	inputIndex *int) (*graphql.ReportConnection, error) {
	slog.Debug("adapter_v2.GetReports",
		"first", first,
	)
	reports, err := a.convenienceService.FindAllByInputIndex(
		ctx,
		first, last, after, before, inputIndex,
	)
	if err != nil {
		slog.Error("Adapter GetReports", "error", err)
		return nil, err
	}
	return a.convertToReportConnection(
		reports.Rows,
		int(reports.Offset),
		int(reports.Total),
	)
}

func (a AdapterV2) convertToReportConnection(
	reports []convenience.Report,
	offset int, total int,
) (*graphql.ReportConnection, error) {
	convNodes := make([]*graphql.Report, len(reports))
	for i := range reports {
		convNodes[i] = a.convertToReport(reports[i])
	}
	return graphql.NewConnection(offset, total, convNodes), nil
}

func (a AdapterV2) convertToReport(
	report convenience.Report,
) *graphql.Report {
	return &graphql.Report{
		Index:      report.Index,
		InputIndex: report.InputIndex,
		Payload:    fmt.Sprintf("0x%s", common.Bytes2Hex(report.Payload)),
	}
}

func (a AdapterV2) GetInputs(
	ctx context.Context,
	first *int,
	last *int,
	after *string,
	before *string,
	where *graphql.InputFilter) (*graphql.InputConnection, error) {

	filters := []*convenience.ConvenienceFilter{}

	inputs, err := a.convenienceService.FindAllInputs(
		ctx,
		first,
		last,
		after,
		before,
		filters,
	)
	if err != nil {
		return nil, err
	}

	return graphql.ConvertToInputConnectionV1(
		inputs.Rows,
		int(inputs.Offset),
		int(inputs.Total),
	)

}

func (a AdapterV2) GetInput(index int) (*graphql.Input, error) {
	ctx := context.Background()
	input, err := a.convenienceService.FindInputByIndex(ctx, index)
	if err != nil {
		return nil, err
	}
	if input == nil {
		return nil, fmt.Errorf("input not found")
	}
	return &graphql.Input{
		Index:               int(input.Index),
		MsgSender:           input.MsgSender.String(),
		Payload:             string(input.Payload),
		Status:              a.convertCompletionStatus(*input),
		Timestamp:           fmt.Sprintf("%d", input.BlockTimestamp.Unix()),
		BlockNumber:         fmt.Sprintf("%d", input.BlockNumber),
		EspressoTimestamp:   fmt.Sprintf("%d", input.EspressoBlockTimestamp.Unix()),
		EspressoBlockNumber: fmt.Sprintf("%d", input.EspressoBlockNumber),
		InputBoxIndex:       input.InputBoxIndex,
	}, nil
}

func (a AdapterV2) GetNotice(noticeIndex int, inputIndex int) (*graphql.Notice, error) {
	ctx := context.Background()
	notice, err := a.convenienceService.FindNoticeByInputAndOutputIndex(
		ctx,
		uint64(inputIndex),
		uint64(noticeIndex),
	)
	if err != nil {
		return nil, err
	}
	if notice == nil {
		return nil, fmt.Errorf("notice not found")
	}
	return &graphql.Notice{
		Index:      noticeIndex,
		InputIndex: inputIndex,
		Payload:    notice.Payload,
	}, nil
}

func (a AdapterV2) GetNotices(
	first *int,
	last *int,
	after *string,
	before *string,
	inputIndex *int) (*graphql.NoticeConnection, error) {
	filters := []*convenience.ConvenienceFilter{}
	if inputIndex != nil {
		field := convenience.INPUT_INDEX
		value := fmt.Sprintf("%d", *inputIndex)
		filters = append(filters, &convenience.ConvenienceFilter{
			Field: &field,
			Eq:    &value,
		})
	}
	ctx := context.Background()
	notices, err := a.convenienceService.FindAllNotices(
		ctx,
		first,
		last,
		after,
		before,
		filters,
	)
	if err != nil {
		return nil, err
	}
	return graphql.ConvertToNoticeConnectionV1(
		notices.Rows,
		int(notices.Offset),
		int(notices.Total),
	)
}

func (a AdapterV2) GetVoucher(voucherIndex int, inputIndex int) (*graphql.Voucher, error) {
	ctx := context.Background()
	voucher, err := a.convenienceService.FindVoucherByInputAndOutputIndex(
		ctx, uint64(inputIndex), uint64(voucherIndex))
	if err != nil {
		return nil, err
	}
	if voucher == nil {
		return nil, fmt.Errorf("voucher not found")
	}
	return &graphql.Voucher{
		Index:       voucherIndex,
		InputIndex:  int(voucher.InputIndex),
		Destination: voucher.Destination.Hex(),
		Payload:     voucher.Payload,
	}, nil
}

func (a AdapterV2) GetVouchers(
	first *int,
	last *int,
	after *string,
	before *string,
	inputIndex *int) (*graphql.VoucherConnection, error) {

	filters := []*convenience.ConvenienceFilter{}
	if inputIndex != nil {
		field := convenience.INPUT_INDEX
		value := fmt.Sprintf("%d", *inputIndex)
		filters = append(filters, &convenience.ConvenienceFilter{
			Field: &field,
			Eq:    &value,
		})
	}
	ctx := context.Background()
	vouchers, err := a.convenienceService.FindAllVouchers(
		ctx,
		first,
		last,
		after,
		before,
		filters,
	)
	if err != nil {
		return nil, err
	}
	return graphql.ConvertToVoucherConnectionV1(
		vouchers.Rows,
		int(vouchers.Offset),
		int(vouchers.Total),
	)
}

func convertReport(node struct {
	Index      int    `json:"index"`
	Blob       string `json:"blob"`
	InputIndex int    `json:"inputIndex"`
}) (*graphql.Report, error) {
	return &graphql.Report{
		Index:      node.Index,
		Payload:    node.Blob,
		InputIndex: node.InputIndex,
	}, nil
}

func (a AdapterV2) convertCompletionStatus(input convenience.AdvanceInput) graphql.CompletionStatus {
	switch input.Status {
	case convenience.CompletionStatusUnprocessed:
		return graphql.CompletionStatusUnprocessed
	case convenience.CompletionStatusAccepted:
		return graphql.CompletionStatusAccepted
	case convenience.CompletionStatusRejected:
		return graphql.CompletionStatusRejected
	case convenience.CompletionStatusException:
		return graphql.CompletionStatusRejected
	default:
		return graphql.CompletionStatusUnprocessed
	}
}
