package audit

import (
	"bitbucket.org/atticlab/go-smart-base/keypair"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/log"
	"encoding/json"
)

type AdminActionSubject string

const (
	SubjectCommission AdminActionSubject = "commission"
)

type ActionPerformed string

const (
	ActionPerformedInsert ActionPerformed = "insert"
	ActionPerformedUpdate ActionPerformed = "update"
	ActionPerformedDelete ActionPerformed = "delete"
)

type AdminActionInfo struct {
	ActorPublicKey  keypair.KP         //public key of the actor, performing task
	Subject         AdminActionSubject //subject to change
	ActionPerformed ActionPerformed    //action performed on subject
	Meta            interface{}        //meta information about audit event
}

func (info *AdminActionInfo) IsValid() bool {
	return info.ActorPublicKey != nil && info.Subject != "" && info.ActionPerformed != ""
}

func (info *AdminActionInfo) ToHistory() *history.AuditLog {
	metaData, err := json.Marshal(info.Meta)
	if err != nil {
		log.WithStack(err).WithError(err).Error("Failed to marshal meta data")
	}
	return &history.AuditLog{
		Actor:   info.ActorPublicKey.Address(),
		Subject: string(info.Subject),
		Action:  string(info.ActionPerformed),
		Meta:    string(metaData),
	}
}
