package adapter

import (
	"fmt"

	"github.com/calindra/nonodo/internal/convenience/model"
)

func ConvertVoucherPayloadToV2(payloadV1 string) string {
	return fmt.Sprintf("0x%s%s", model.VOUCHER_SELECTOR, payloadV1)
}

func ConvertNoticePayloadToV2(payloadV1 string) string {
	return fmt.Sprintf("0x%s%s", model.NOTICE_SELECTOR, payloadV1)
}
