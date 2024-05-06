package reader

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

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

func NewAdapterV2(
	convenienceService *services.ConvenienceService,
	httpClient HttpClient,
) Adapter {
	slog.Debug("NewAdapterV2")
	return AdapterV2{
		convenienceService: convenienceService,
		httpClient:         httpClient,
	}
}

func (a AdapterV2) GetReport(reportIndex int, inputIndex int) (*graphql.Report, error) {
	requestBody := []byte(fmt.Sprintf(`{
    "query": "query Reports($index: Int, $inputIndex: Int) { reports(condition: {index: $index, inputIndex: $inputIndex}) { edges { node { index blob inputIndex } } } }",
    "variables": {
      "index": %d,
      "inputIndex": %d
    }
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

func (a AdapterV2) GetReports(first *int, last *int, after *string, before *string, inputIndex *int) (*model.ReportConnection, error) {
	forward := first != nil || after != nil
	backward := last != nil || before != nil

	if forward && backward {
		return nil, commons.ErrMixedPagination
	}

	if !forward && !backward {
		// If nothing was set, use forward pagination by default
		forward = true
	}

	if forward {
		var afterValue string

		if after != nil {
			afterValue = *after
		}

		requestBody := []byte(fmt.Sprintf(`{
			"query": "query MyQuery($first: Int, $after: String, $inputIndex: Int) { reports(first: $first, after: $after, condition: {inputIndex: $inputIndex}) { edges { cursor node { index inputIndex blob } } } }",
			"variables": {
			  "first": %d,
			  "after": %s,
              "inputIndex": %d
			}
		  }`, first, afterValue, inputIndex))

		response, err := a.httpClient.Post(requestBody)

		if err != nil {
			slog.Error("Error calling Graphile Reports", "error", err)
			return nil, err
		}

		var reportByIdResponse ReportByIdResponse
		err = json.Unmarshal(response, &reportByIdResponse)

		if err != nil {
			slog.Error("Error decoding JSON", "error", err)
			return nil, err
		}

		reports := make([]*graphql.Report, len(reportByIdResponse.Data.Reports.Edges))

		for i := range reportByIdResponse.Data.Reports.Edges {
			convertedReport, err := convertReport(reportByIdResponse.Data.Reports.Edges[i].Node)

			if err != nil {
				return nil, err
			}

			reports = append(reports, convertedReport)
		}

		var offset int

		if after != nil {
			offset, err = commons.DecodeCursor(*after, len(reportByIdResponse.Data.Reports.Edges))
			if err != nil {
				return nil, err
			}
			offset = offset + 1
		} else {
			offset = 0
		}

		return graphql.NewConnection(offset, len(reportByIdResponse.Data.Reports.Edges), reports), nil

	} else {
		var beforeValue string

		if before != nil {
			beforeValue = *before
		}

		requestBody := []byte(fmt.Sprintf(`{
			"query": "query MyQuery($last: Int, $before: String, $inputIndex: Int) { reports(last: $last, before: $before, condition: {inputIndex: $inputIndex}) { edges { cursor node { index inputIndex blob } } } }",
			"variables": {
			  "last": %d,
			  "before": %s,
              "inputIndex": %d
			}
		  }`, last, beforeValue, inputIndex))

		response, err := a.httpClient.Post(requestBody)

		if err != nil {
			slog.Error("Error calling Graphile Reports", "error", err)
			return nil, err
		}

		var reportByIdResponse ReportByIdResponse
		err = json.Unmarshal(response, &reportByIdResponse)

		if err != nil {
			slog.Error("Error decoding JSON", "error", err)
			return nil, err
		}

		reports := make([]*graphql.Report, len(reportByIdResponse.Data.Reports.Edges))

		for i := range reportByIdResponse.Data.Reports.Edges {
			convertedReport, err := convertReport(reportByIdResponse.Data.Reports.Edges[i].Node)

			if err != nil {
				return nil, err
			}

			reports = append(reports, convertedReport)
		}

		var beforeOffset int
		total := len(reportByIdResponse.Data.Reports.Edges)

		var limit int
		if last != nil {
			if *last < 0 {
				return nil, commons.ErrInvalidLimit
			}
			limit = *last
		} else {
			limit = commons.DefaultPaginationLimit
		}

		var offset int

		if before != nil {
			beforeOffset, err = commons.DecodeCursor(*before, total)
			if err != nil {
				return nil, err
			}
		} else {
			beforeOffset = total
		}
		offset = max(0, beforeOffset-limit)

		return graphql.NewConnection(offset, len(reportByIdResponse.Data.Reports.Edges), reports), nil

	}
}

func (a AdapterV2) GetInputs(first *int, last *int, after *string, before *string, where *model.InputFilter) (*model.InputConnection, error) {
	forward := first != nil || after != nil
	backward := last != nil || before != nil

	if forward && backward {
		return nil, commons.ErrMixedPagination
	}

	if !forward && !backward {
		// If nothing was set, use forward pagination by default
		forward = true
	}

	if forward {
		var afterValue string

		if after != nil {
			afterValue = *after
		}

		requestBody := []byte(fmt.Sprintf(`{
		"query": "query MyQuery($first: Int, $after: String) { inputs(first: $first, after: $after) { edges { cursor node { index blob status } } } }",
		"variables": {
		  "first": %d,
		  "after": %s
		}
	  }`, first, afterValue))

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

		inputs := make([]*graphql.Input, len(inputByIdResponse.Data.Inputs.Edges))

		for i := range inputByIdResponse.Data.Inputs.Edges {
			convertedInput, err := convertInput(inputByIdResponse.Data.Inputs.Edges[i].Node)

			if err != nil {
				return nil, err
			}

			inputs = append(inputs, convertedInput)
		}

		var offset int

		if after != nil {
			offset, err = commons.DecodeCursor(*after, len(inputByIdResponse.Data.Inputs.Edges))
			if err != nil {
				return nil, err
			}
			offset = offset + 1
		} else {
			offset = 0
		}

		return graphql.NewConnection(offset, len(inputByIdResponse.Data.Inputs.Edges), inputs), nil

	} else {
		var beforeValue string

		if before != nil {
			beforeValue = *before
		}

		requestBody := []byte(fmt.Sprintf(`{
		"query": "query MyQuery($last: Int, $before: String) { inputs(last: $last, before: $before) { edges { cursor node { index blob status } } } }",
		"variables": {
		  "last": %d,
		  "before": %s
		}
	  }`, last, beforeValue))

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

		inputs := make([]*graphql.Input, len(inputByIdResponse.Data.Inputs.Edges))

		for i := range inputByIdResponse.Data.Inputs.Edges {
			convertedInput, err := convertInput(inputByIdResponse.Data.Inputs.Edges[i].Node)

			if err != nil {
				return nil, err
			}

			inputs = append(inputs, convertedInput)
		}

		var beforeOffset int
		total := len(inputByIdResponse.Data.Inputs.Edges)

		var limit int
		if last != nil {
			if *last < 0 {
				return nil, commons.ErrInvalidLimit
			}
			limit = *last
		} else {
			limit = commons.DefaultPaginationLimit
		}

		var offset int

		if before != nil {
			beforeOffset, err = commons.DecodeCursor(*before, total)
			if err != nil {
				return nil, err
			}
		} else {
			beforeOffset = total
		}
		offset = max(0, beforeOffset-limit)

		return graphql.NewConnection(offset, len(inputByIdResponse.Data.Inputs.Edges), inputs), nil
	}
}

func (a AdapterV2) GetInput(index int) (*graphql.Input, error) {
	requestBody := []byte(fmt.Sprintf(`{
        "query": "query Inputs($index: Int) { inputs(condition: {index: $index}) { edges { node { index blob status } } } }",
        "variables": {
            "index":%d
        }
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
		return convertInput(inputByIdResponse.Data.Inputs.Edges[0].Node)
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
		Proof:      nil,
	}, nil
}

func (a AdapterV2) GetNotices(first *int, last *int, after *string, before *string, inputIndex *int) (*model.NoticeConnection, error) {
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

func (a AdapterV2) GetVouchers(first *int, last *int, after *string, before *string, inputIndex *int) (*model.VoucherConnection, error) {
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

// finalize convert
func convertInput(node struct {
	Index  int    `json:"index"`
	Blob   string `json:"blob"`
	Status string `json:"status"`
}) (*graphql.Input, error) {
	return &graphql.Input{
		Index:       node.Index,
		Status:      convertCompletionStatus(node.Status),
		MsgSender:   "",
		Timestamp:   "",
		BlockNumber: "",
		Payload:     "",
	}, nil
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

func convertCompletionStatus(status string) graphql.CompletionStatus {
	switch status {
	case model.CompletionStatusUnprocessed.String():
		return graphql.CompletionStatusUnprocessed
	case model.CompletionStatusAccepted.String():
		return graphql.CompletionStatusAccepted
	case model.CompletionStatusRejected.String():
		return graphql.CompletionStatusRejected
	case model.CompletionStatusException.String():
		return graphql.CompletionStatusException
	default:
		panic("invalid completion status")
	}
}
