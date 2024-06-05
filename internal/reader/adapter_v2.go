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
	repos "github.com/calindra/nonodo/internal/model"
	"github.com/calindra/nonodo/internal/reader/model"
	graphql "github.com/calindra/nonodo/internal/reader/model"
)

type AdapterV2 struct {
	convenienceService *services.ConvenienceService
	httpClient         HttpClient
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
	httpClient HttpClient,
	inputBlobAdapter InputBlobAdapter,
) Adapter {
	slog.Debug("NewAdapterV2")
	return AdapterV2{
		convenienceService: convenienceService,
		httpClient:         httpClient,
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
	response, err := a.httpClient.Post(payload)
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

	response, err := a.httpClient.Post(requestBody)

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

	response, err := a.httpClient.Post(requestBody)

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
		requestBody, _ = getInputForwardQuery(first, after, where)
	} else {
		requestBody, _ = getInputBackwardQuery(last, before, where)
	}

	response, err := a.httpClient.Post(requestBody)

	if err != nil {
		slog.Error("Error calling Graphile Inputs", "error", err)
		return nil, err
	}

	return a.processInputsResponse(response, after, before, forward, last)
}

func getInputBackwardQuery(last *int, before *string, where *graphql.InputFilter) ([]byte, error) {
	var builder strings.Builder

	builder.WriteString(`{ "query": "query { inputs(`)

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

	if where != nil {
		if paramsAdded {
			builder.WriteString(", ")
		}
		builder.WriteString("filter: {")

		filterParamsAdded := false

		if where.IndexLowerThan != nil {
			builder.WriteString(fmt.Sprintf("index: {lessThan: %d}", *where.IndexLowerThan))
			filterParamsAdded = true
		}

		if where.IndexGreaterThan != nil {
			if filterParamsAdded {
				builder.WriteString(", ")
			}
			builder.WriteString(fmt.Sprintf("index: {greaterThan: %d}", *where.IndexGreaterThan))
			filterParamsAdded = true
		}

		builder.WriteString("}")
	}

	builder.WriteString(`) { edges { cursor node { index blob status }}} }" }`)

	return []byte(builder.String()), nil
}

func getInputForwardQuery(first *int, after *string, where *graphql.InputFilter) ([]byte, error) {
	var builder strings.Builder

	builder.WriteString(`{ "query": "query { inputs(`)

	paramsAdded := false

	if first != nil {
		builder.WriteString(fmt.Sprintf("first: %d", *first))
		paramsAdded = true
	}

	if after != nil {
		if paramsAdded {
			builder.WriteString(", ")
		}
		builder.WriteString(fmt.Sprintf("after: \"%s\"", *after))
		paramsAdded = true
	}

	if where != nil {
		if paramsAdded {
			builder.WriteString(", ")
		}
		builder.WriteString("filter: {")

		filterParamsAdded := false

		if where.IndexLowerThan != nil {
			builder.WriteString(fmt.Sprintf("index: {lessThan: %d}", *where.IndexLowerThan))
			filterParamsAdded = true
		}

		if where.IndexGreaterThan != nil {
			if filterParamsAdded {
				builder.WriteString(", ")
			}
			builder.WriteString(fmt.Sprintf("index: {greaterThan: %d}", *where.IndexGreaterThan))
			filterParamsAdded = true
		}

		builder.WriteString("}")
	}

	builder.WriteString(`) { edges { cursor node { index blob status }}} }" }`)

	return []byte(builder.String()), nil
}

func (a AdapterV2) GetInput(index int) (*graphql.Input, error) {
	slog.Info(fmt.Sprintf("Adapter V2 - GetInput %d", index))

	requestBody := []byte(fmt.Sprintf(`
        {
			"query": "query { inputs(condition: {index: %d}) { edges { node { index blob status}}}}"
			
    	}`, index))

	response, err := a.httpClient.Post(requestBody)

	if err != nil {
		slog.Error("Error calling Graphile Inouts", "error", err)
		return nil, err
	}

	var inputByIdResponse InputByIdResponse
	err = json.Unmarshal(response, &inputByIdResponse)

	if err != nil {
		slog.Error("Error decoding JSON", "error", err)
		return nil, err
	}

	if len(inputByIdResponse.Data.Inputs.Edges) > 0 {
		return a.InputBlobAdapter.Adapt(inputByIdResponse.Data.Inputs.Edges[0].Node)
	}

	return nil, nil
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
		field := repos.INPUT_INDEX
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
		field := repos.INPUT_INDEX
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

func (a AdapterV2) processInputsResponse(response []byte, after *string, before *string, forward bool, last *int) (*model.InputConnection, error) {
	var inputByIdResponse InputByIdResponse
	err := json.Unmarshal(response, &inputByIdResponse)
	if err != nil {
		slog.Error("Error decoding JSON", "error", err)
		return nil, err
	}

	inputs := make([]*graphql.Input, 0, len(inputByIdResponse.Data.Inputs.Edges))
	for _, edge := range inputByIdResponse.Data.Inputs.Edges {
		convertedInput, err := a.InputBlobAdapter.Adapt(edge.Node)
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, convertedInput)
	}

	if forward {
		offset, err := calculateOffset(after, len(inputByIdResponse.Data.Inputs.Edges))
		if err != nil {
			return nil, err
		}
		return graphql.NewConnection(offset, len(inputByIdResponse.Data.Inputs.Edges), inputs), nil
	} else {
		offset, err := calculateOffsetBefore(before, len(inputByIdResponse.Data.Inputs.Edges), last)
		if err != nil {
			return nil, err
		}
		return graphql.NewConnection(offset, len(inputByIdResponse.Data.Inputs.Edges), inputs), nil
	}
}
