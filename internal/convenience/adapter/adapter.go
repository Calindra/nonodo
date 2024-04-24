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

// for a while we will remove the prefix
// until the v2 does not arrives
func RemoveSelector(payload string) string {
	return fmt.Sprintf("0x%s", payload[10:])
}
