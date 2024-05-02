package reader

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/calindra/nonodo/internal/commons"
	convenience "github.com/calindra/nonodo/internal/convenience/model"
	"github.com/calindra/nonodo/internal/convenience/services"
	repos "github.com/calindra/nonodo/internal/model"
	"github.com/calindra/nonodo/internal/reader/model"
	graphql "github.com/calindra/nonodo/internal/reader/model"
	"github.com/jmoiron/sqlx"
	"io"
	"log/slog"
	"net/http"
)

type AdapterV2 struct {
	reportRepository   *repos.ReportRepository
	inputRepository    *repos.InputRepository
	convenienceService *services.ConvenienceService
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
	db *sqlx.DB,
	convenienceService *services.ConvenienceService,
) Adapter {
	slog.Debug("NewAdapterV2")
	reportRepository := &repos.ReportRepository{
		Db: db,
	}
	err := reportRepository.CreateTables()
	if err != nil {
		panic(err)
	}
	inputRepository := &repos.InputRepository{
		Db: db,
	}
	err = inputRepository.CreateTables()
	if err != nil {
		panic(err)
	}
	return AdapterV2{
		reportRepository:   reportRepository,
		inputRepository:    inputRepository,
		convenienceService: convenienceService,
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

	req, err := http.NewRequest("POST", "http://localhost:5000/graphql", bytes.NewBuffer(requestBody))
	if err != nil {
		slog.Error("Error creating request", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		fmt.Println("Error executing request:", err)
		return nil, err
	}

	defer resp.Body.Close()

	// Lê o corpo da resposta
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Erro ao ler o corpo da resposta:", err)
		return nil, err
	}

	var reportByIdResponse ReportByIdResponse
	err = json.Unmarshal(body, &reportByIdResponse)

	if err != nil {
		fmt.Println("Erro ao decodificar JSON:", err)
		return nil, err
	}

	if len(reportByIdResponse.Data.Reports.Edges) > 0 {
		return convertReport(reportByIdResponse)
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
		//TODO setar query aqui
		requestBody := []byte(fmt.Sprintf(``, first, after))

		req, err := http.NewRequest("POST", "http://localhost:5000/graphql", bytes.NewBuffer(requestBody))

	} else {
		//TODO setar query aqui
		requestBody := []byte(fmt.Sprintf(``, last, before))

		req, err := http.NewRequest("POST", "http://localhost:5000/graphql", bytes.NewBuffer(requestBody))
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
		//TODO setar query aqui
		requestBody := []byte(fmt.Sprintf(``, first, after))

		req, err := http.NewRequest("POST", "http://localhost:5000/graphql", bytes.NewBuffer(requestBody))

	} else {
		//TODO setar query aqui
		requestBody := []byte(fmt.Sprintf(``, last, before))

		req, err := http.NewRequest("POST", "http://localhost:5000/graphql", bytes.NewBuffer(requestBody))
	}
}

func (a AdapterV2) GetInput(index int) (*graphql.Input, error) {
	requestBody := []byte(fmt.Sprintf(`{
        "query": "query Inputs($index: Int) { inputs(condition: {index: $index}) { edges { node { index blob status } } } }",
        "variables": {
            "index":%d
        }
    }`, index))

	req, err := http.NewRequest("POST", "http://localhost:5000/graphql", bytes.NewBuffer(requestBody))
	if err != nil {
		slog.Error("Error creating request", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		fmt.Println("Error executing request:", err)
		return nil, err
	}

	defer resp.Body.Close()

	// Lê o corpo da resposta
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Erro ao ler o corpo da resposta:", err)
		return nil, err
	}

	var inputByIdResponse InputByIdResponse
	err = json.Unmarshal(body, &inputByIdResponse)

	if err != nil {
		fmt.Println("Erro ao decodificar JSON:", err)
		return nil, err
	}

	if len(inputByIdResponse.Data.Inputs.Edges) > 0 {
		return convertInput(inputByIdResponse)
	}

	return nil, nil

}

func (a AdapterV2) GetNotice(noticeIndex int, inputIndex int) (*model.Notice, error) {
	ctx := context.Background()
	notice, err := a.convenienceService.FindVoucherByInputAndOutputIndex(
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

func convertInput(response InputByIdResponse) (*graphql.Input, error) {
	//TODO completar conversão
	node := response.Data.Inputs.Edges[0].Node
	return &graphql.Input{
		Index:       node.Index,
		Status:      convertCompletionStatus(node.Status),
		MsgSender:   "",
		Timestamp:   "",
		BlockNumber: "",
		Payload:     node.Blob,
	}, nil
}

func convertReport(response ReportByIdResponse) (*graphql.Report, error) {
	node := response.Data.Reports.Edges[0].Node
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
