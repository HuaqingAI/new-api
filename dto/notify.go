package dto

import relaydto "github.com/QuantumNous/new-api/relaykit/dto"

type Notify = relaydto.Notify

const ContentValueParam = relaydto.ContentValueParam

const (
	NotifyTypeQuotaExceed   = relaydto.NotifyTypeQuotaExceed
	NotifyTypeChannelUpdate = relaydto.NotifyTypeChannelUpdate
	NotifyTypeChannelTest   = relaydto.NotifyTypeChannelTest
)

var NewNotify = relaydto.NewNotify
