package maccommand

import (
	"fmt"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/brocaar/loraserver/internal/common"
	"github.com/brocaar/loraserver/internal/storage"
	"github.com/brocaar/lorawan"
)

// RequestPingSlotChannel modifies the frequency and / or the data-rate
// on which the end-device expects the downlink pings (class-b).
func RequestPingSlotChannel(devEUI lorawan.EUI64, dr, freq int) error {
	block := Block{
		CID: lorawan.PingSlotChannelReq,
		MACCommands: []lorawan.MACCommand{
			{
				CID: lorawan.PingSlotChannelReq,
				Payload: &lorawan.PingSlotChannelReqPayload{
					Frequency: uint32(freq / 100),
					DR:        uint8(dr),
				},
			},
		},
	}

	if err := AddQueueItem(common.RedisPool, devEUI, block); err != nil {
		return errors.Wrap(err, "add mac-command queue item error")
	}

	return nil
}

func handlePingSlotChannelAns(ds *storage.DeviceSession, block Block, pendingBlock *Block) error {
	if len(block.MACCommands) != 1 {
		return fmt.Errorf("exactly one mac-command expected, got: %d", len(block.MACCommands))
	}

	if pendingBlock == nil || len(pendingBlock.MACCommands) == 0 {
		return ErrDoesNotExist
	}
	req := pendingBlock.MACCommands[0].Payload.(*lorawan.PingSlotChannelReqPayload)

	pl, ok := block.MACCommands[0].Payload.(*lorawan.PingSlotChannelAnsPayload)
	if !ok {
		return fmt.Errorf("expected *lorawan.PingSlotChannelAnsPayload, got %T", block.MACCommands[0].Payload)
	}

	if !pl.ChannelFrequencyOK || !pl.DataRateOK {
		log.WithFields(log.Fields{
			"channel_frequency_ok": pl.ChannelFrequencyOK,
			"data_rate_ok":         pl.DataRateOK,
		}).Warning("ping_slot_channel request not acknowledged")
		return nil
	}

	ds.PingSlotDR = int(req.DR)
	ds.PingSlotFrequency = int(req.Frequency) * 100

	log.WithFields(log.Fields{
		"channel_frequency": ds.PingSlotFrequency,
		"data_rate":         ds.PingSlotDR,
	}).Info("ping_slot_channel request acknowledged")

	return nil
}
