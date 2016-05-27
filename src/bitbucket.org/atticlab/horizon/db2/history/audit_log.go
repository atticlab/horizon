package history

import (
	sq "github.com/lann/squirrel"
)

// CreateAuditLogEntry adds row to audit_log
// actorPublicKey - public key of the actor, performing task
// subject - subject to change
// action - action performed on subject
// meta - meta information about audit event
func (q *Q) CreateAuditLogEntry(
    actorPublicKey string,
    subject string,
    action string,
    meta string, 
) error {
    sql := createAuditLogEntry.Values(actorPublicKey, subject, action, meta)
    _, err := q.Exec(sql)
    
    return err
}

var createAuditLogEntry = sq.Insert("audit_log").Columns(
	"actor",
    "subject",
    "action",
    "meta",
)
