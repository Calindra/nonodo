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
		Proof struct {
			InputIndex int `json:"inputIndex"`
		}
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

	return &graphql.Proof{
		InputIndex: theProof.Data.Proof.InputIndex,
	}, nil
}

func (a AdapterV2) GetReport(reportIndex int, inputIndex int) (*graphql.Report, error) {
	requestBody := []byte(fmt.Sprintf(`
    {
        "query": "query Reports($index: Int, $inputIndex: Int) {
            reports(condition: {index: $index, inputIndex: $inputIndex}) {
                edges {
                    node {
                        index
                        blob
                        inputIndex
                    }
                }
            }
        }",
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

	if forward {
		var afterValue string

		if after != nil {
			afterValue = *after
		}

		requestBody := []byte(fmt.Sprintf(`
    {
        "query": "query MyQuery($first: Int, $after: String, $inputIndex: Int) {
            reports(first: $first, after: $after, condition: {inputIndex: $inputIndex}) {
                edges {
                    cursor
                    node {
                        index
                        inputIndex
                        blob
                    }
                }
            }
        }",
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
			convertedReport, err :=
				convertReport(
					reportByIdResponse.Data.Reports.Edges[i].Node)

			if err != nil {
				return nil, err
			}

			reports = append(reports, convertedReport)
		}

		var offset int

		if after != nil {
			offset, err = commons.DecodeCursor(
				*after,
				len(reportByIdResponse.Data.Reports.Edges))

			if err != nil {
				return nil, err
			}
			offset = offset + 1
		} else {
			offset = 0
		}

		return graphql.NewConnection(
			offset,
			len(reportByIdResponse.Data.Reports.Edges),
			reports), nil

	} else {
		var beforeValue string

		if before != nil {
			beforeValue = *before
		}

		requestBody := []byte(fmt.Sprintf(`
    {
        "query": "query MyQuery($last: Int, $before: String, $inputIndex: Int) {
            reports(last: $last, before: $before, condition: {inputIndex: $inputIndex}) {
                edges {
                    cursor
                    node {
                        index
                        inputIndex
                        blob
                    }
                }
            }
        }",
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
			convertedReport, err :=
				convertReport(
					reportByIdResponse.Data.Reports.Edges[i].Node)

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

		return graphql.NewConnection(
			offset,
			len(reportByIdResponse.Data.Reports.Edges),
			reports), nil

	}
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

	if forward {
		query := getInputForwardQuery(first, after, where)

		requestBody := []byte(query)

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
			convertedInput, err :=
				a.InputBlobAdapter.Adapt(
					inputByIdResponse.Data.Inputs.Edges[i].Node)

			if err != nil {
				return nil, err
			}

			inputs = append(inputs, convertedInput)
		}

		var offset int

		if after != nil {
			offset, err = commons.DecodeCursor(
				*after,
				len(inputByIdResponse.Data.Inputs.Edges))

			if err != nil {
				return nil, err
			}
			offset = offset + 1
		} else {
			offset = 0
		}

		return graphql.NewConnection(
			offset,
			len(inputByIdResponse.Data.Inputs.Edges),
			inputs), nil

	} else {
		query := getInputBackwardQuery(last, before, where)

		requestBody := []byte(query)

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
			convertedInput, err :=
				a.InputBlobAdapter.Adapt(
					inputByIdResponse.Data.Inputs.Edges[i].Node)

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

		return graphql.NewConnection(
			offset,
			len(inputByIdResponse.Data.Inputs.Edges),
			inputs), nil
	}
}

func getInputBackwardQuery(last *int, before *string, where *graphql.InputFilter) string {
	var beforeValue string

	if before != nil {
		beforeValue = *before
	}

	if where != nil {
		if where.IndexLowerThan != nil {
			return fmt.Sprintf(`
    {
        "query": "query MyQuery($filter: InputFilter, $last: Int, $before: String) {
            inputs(filter: $filter, last: $last, before: $before) {
                edges {
                    cursor
                    node {
                        index
                        blob
                        status
                    }
                }
            }
        }",
        "variables": {
            "filter": {
                "index": {
                    "lessThan": %d
                }
            },
            "last": %d,
            "before": %s
        }
    }`, *where.IndexLowerThan, last, beforeValue)

		}

		if where.IndexGreaterThan != nil {
			return fmt.Sprintf(`
    {
        "query": "query MyQuery($filter: InputFilter, $last: Int, $before: String) {
            inputs(filter: $filter, last: $last, before: $before) {
                edges {
                    cursor
                    node {
                        index
                        blob
                        status
                    }
                }
            }
        }",
        "variables": {
            "filter": {
                "index": {
                    "greaterThan": %d
                }
            },
            "last": %d,
            "before": %s
        }
    }`, *where.IndexGreaterThan, last, beforeValue)

		}
	}

	return fmt.Sprintf(`
    {
        "query": "query MyQuery($last: Int, $before: String) {
            inputs(last: $last, before: $before) {
                edges {
                    cursor
                    node {
                        index
                        blob
                        status
                    }
                }
            }
        }",
        "variables": {
            "last": %d,
            "before": %s
        }
    }`, last, beforeValue)

}

func getInputForwardQuery(first *int, after *string, where *graphql.InputFilter) string {
	var afterValue string

	if after != nil {
		afterValue = *after
	}

	if where != nil {
		if where.IndexLowerThan != nil {
			return fmt.Sprintf(`
    {
        "query": "query MyQuery($filter: InputFilter, $first: Int, $after: String) {
            inputs(filter: $filter, first: $first, after: $after) {
                edges {
                    cursor
                    node {
                        index
                        blob
                        status
                    }
                }
            }
        }",
        "variables": {
            "filter": {
                "index": {
                    "lessThan": %d
                }
            },
            "first": %d,
            "after": %s
        }
    }`, *where.IndexLowerThan, first, afterValue)
		}

		if where.IndexGreaterThan != nil {
			return fmt.Sprintf(`
    {
        "query": "query MyQuery($filter: InputFilter, $first: Int, $after: String) {
            inputs(filter: $filter, first: $first, after: $after) {
                edges {
                    cursor
                    node {
                        index
                        blob
                        status
                    }
                }
            }
        }",
        "variables": {
            "filter": {
                "index": {
                    "greaterThan": %d
                }
            },
            "first": %d,
            "after": %s
        }
    }`, *where.IndexGreaterThan, first, afterValue)
		}
	}

	return fmt.Sprintf(`
	   {
			"query": "query MyQuery($first: Int, $after: String) {
				inputs(first: $first, after: $after) {
					edges {
						cursor
						node {
							index
							blob
							status
						}
					}
				}
			}",
			"variables": {
				"first": %d,
				"after": %s
			}
		}`, first, afterValue)
}

func (a AdapterV2) GetInput(index int) (*graphql.Input, error) {
	requestBody := []byte(fmt.Sprintf(`{
       {
			"query": "query Inputs($index: Int) { 
				inputs(condition: {index: $index}) {
					edges {
						node {
							index
							blob
							status
						}
					}
				}
			}",
			"variables": {
				"index": %d
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
		// Proof:      nil,
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
