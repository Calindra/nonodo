package reader

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/calindra/nonodo/internal/commons"
	convenience "github.com/calindra/nonodo/internal/convenience/model"
	"github.com/calindra/nonodo/internal/convenience/services"
	"github.com/calindra/nonodo/internal/graphile"
	"github.com/calindra/nonodo/internal/reader/model"
	graphql "github.com/calindra/nonodo/internal/reader/model"
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
	first *int,
	last *int,
	after *string,
	before *string,
	inputIndex *int) (*model.ReportConnection, error) {

	forward := first != nil || after != nil
	backward := last != nil || before != nil

	if forward && backward {
		return nil, commons.ErrMixedPagination
	}

	if !forward && !backward {
		// If nothing was set, use forward pagination by default
		forward = true
	}

	var requestBody []byte
	var err error

	if forward {
		requestBody, _ = createForwardRequestBody(first, after, inputIndex)
	} else {
		requestBody, _ = createBackwardRequestBody(last, before, inputIndex)
	}

	response, err := a.graphileClient.Post(requestBody)

	if err != nil {
		slog.Error("Error calling Graphile Reports", "error", err)
		return nil, err
	}

	return processReportsResponse(response, after, before, forward, last)
}

func (a AdapterV2) GetInputs(
	first *int,
	last *int,
	after *string,
	before *string,
	where *model.InputFilter) (*model.InputConnection, error) {

	filters := []*convenience.ConvenienceFilter{}

	ctx := context.Background()
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
		Index:       int(input.Index),
		MsgSender:   input.MsgSender.String(),
		Payload:     string(input.Payload),
		Status:      a.convertCompletionStatus(*input),
		Timestamp:   fmt.Sprintf("%d", input.BlockTimestamp.Unix()),
		BlockNumber: fmt.Sprintf("%d", input.BlockNumber),
	}, nil
}

func (a AdapterV2) GetNotice(noticeIndex int, inputIndex int) (*model.Notice, error) {
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
	inputIndex *int) (*model.NoticeConnection, error) {
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

func (a AdapterV2) GetVoucher(voucherIndex int, inputIndex int) (*model.Voucher, error) {
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
	inputIndex *int) (*model.VoucherConnection, error) {

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

func createForwardRequestBody(first *int, after *string, inputIndex *int) ([]byte, error) {
	var builder strings.Builder

	builder.WriteString(`{ "query": "query { reports(`)

	if first != nil {
		builder.WriteString(fmt.Sprintf("first: %d", *first))
	}

	if after != nil {
		if first != nil {
			builder.WriteString(", ")
		}
		builder.WriteString(fmt.Sprintf("after: \"%s\"", *after))
	}

	if inputIndex != nil {
		if first != nil || after != nil {
			builder.WriteString(", ")
		}
		builder.WriteString(fmt.Sprintf("condition: {inputIndex: %d}", *inputIndex))
	}

	builder.WriteString(`) { edges { cursor node { index inputIndex blob }}} }" }`)

	return []byte(builder.String()), nil
}

func createBackwardRequestBody(last *int, before *string, inputIndex *int) ([]byte, error) {
	var builder strings.Builder

	builder.WriteString(`{ "query": "query { reports(`)

	paramsAdded := false

	if last != nil {
		builder.WriteString(fmt.Sprintf("last: %d", *last))
		paramsAdded = true
	}

	if before != nil {
		if paramsAdded {
			builder.WriteString(", ")
		}
		builder.WriteString(fmt.Sprintf("before: \"%s\"", *before))
		paramsAdded = true
	}

	if inputIndex != nil {
		if paramsAdded {
			builder.WriteString(", ")
		}
		builder.WriteString(fmt.Sprintf("condition: {inputIndex: %d}", *inputIndex))
	}

	builder.WriteString(`) { edges { cursor node { index inputIndex blob }}} }" }`)

	return []byte(builder.String()), nil
}

func processReportsResponse(response []byte, after *string, before *string, forward bool, last *int) (*model.ReportConnection, error) {
	var reportByIdResponse ReportByIdResponse
	err := json.Unmarshal(response, &reportByIdResponse)
	if err != nil {
		slog.Error("Error decoding JSON", "error", err)
		return nil, err
	}

	reports := make([]*graphql.Report, 0, len(reportByIdResponse.Data.Reports.Edges))
	for _, edge := range reportByIdResponse.Data.Reports.Edges {
		convertedReport, err := convertReport(edge.Node)
		if err != nil {
			return nil, err
		}
		reports = append(reports, convertedReport)
	}

	if forward {
		offset, err := calculateOffset(after, len(reportByIdResponse.Data.Reports.Edges))
		if err != nil {
			return nil, err
		}
		return graphql.NewConnection(offset, len(reportByIdResponse.Data.Reports.Edges), reports), nil
	} else {
		offset, err := calculateOffsetBefore(before, len(reportByIdResponse.Data.Reports.Edges), last)
		if err != nil {
			return nil, err
		}
		return graphql.NewConnection(offset, len(reportByIdResponse.Data.Reports.Edges), reports), nil
	}
}

func calculateOffset(after *string, length int) (int, error) {
	if after != nil {
		offset, err := commons.DecodeCursor(*after, length)
		if err != nil {
			return 0, err
		}
		return offset + 1, nil
	}
	return 0, nil
}

func calculateOffsetBefore(before *string, length int, last *int) (int, error) {
	var beforeOffset int
	if before != nil {
		offset, err := commons.DecodeCursor(*before, length)
		if err != nil {
			return 0, err
		}
		beforeOffset = offset
	} else {
		beforeOffset = length
	}

	var limit int
	if last != nil {
		if *last < 0 {
			return 0, commons.ErrInvalidLimit
		}
		limit = *last
	} else {
		limit = commons.DefaultPaginationLimit
	}

	return max(0, beforeOffset-limit), nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
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
