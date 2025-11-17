package processor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"cloud.google.com/go/pubsub"
	"github.com/censys/scan-takehome/pkg/shared"
)

func Listen(sub *pubsub.Subscription, store Store) {
	errReceive := sub.Receive(context.Background(), func(ctx context.Context, m *pubsub.Message) {
		var scan shared.Scan
		if errUnmarshal := json.Unmarshal(m.Data, &scan); errUnmarshal != nil {
			log.Printf("Unable to unmarshal incoming message. Rejecting invalid message silently. error: %v data: %v", errUnmarshal, m.Data)
			m.Ack()
		} else {
			errProcess := process(ctx, &scan, store)
			if errProcess != nil {
				m.Nack()
			} else {
				m.Ack()
			}
		}
	})

	if errReceive != nil && !errors.Is(errReceive, context.Canceled) {
		panic(fmt.Sprintf("sub.Receive failed unexpectedly %v", errReceive))
	}

	log.Printf("\nEND")
}
