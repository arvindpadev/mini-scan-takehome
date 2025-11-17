package processor

import (
	"context"
	"encoding/base64"
	"log"

	"github.com/censys/scan-takehome/pkg/shared"
)

func process(context context.Context, scan *shared.Scan, store Store) error {
	log.Printf("Got scan: %v\n", scan)

	var data string
	if scan.DataVersion == shared.V1 {
		v1DataMap := scan.Data.(map[string]interface{})
		encoded := v1DataMap["response_bytes_utf8"].(string)
		decoded, errDecode := base64.StdEncoding.DecodeString(encoded)
		if errDecode != nil {
			log.Printf("The data is invalid for %v. Rejecting silently %v", scan, errDecode)
			return nil
		}

		log.Printf("For scan %v, the data received was '%s'", scan, data)
		data = string(decoded)
	} else {
		v2DataMap := scan.Data.(map[string]interface{})
		data = v2DataMap["response_str"].(string)
	}

	return store.StoreScan(context, &StorableScan{
		ip:        scan.Ip,
		port:      scan.Port,
		service:   scan.Service,
		timestamp: scan.Timestamp,
		data:      data,
	})
}
