package core

import (
	sq "github.com/lann/squirrel"
)

type StatisticsQ struct {
	Err    error
	parent *Q
	sql    sq.SelectBuilder
}

func (q *Q) Statistics() *StatisticsQ {
	return &StatisticsQ{
		parent: q,
		sql:    selectStats,
	}
}

func (q *StatisticsQ) ForAccount(aid string) *StatisticsQ {
	if q.Err != nil {
		return q
	}

	q.sql = q.sql.Where("stat.account_id = ?", aid)
	return q
}

func (q *StatisticsQ) ForAssetCode(code string) *StatisticsQ {
	if q.Err != nil {
		return q
	}

	q.sql = q.sql.Where("stat.asset_code = ?", code)
	return q
}

func (q *StatisticsQ) ForAssetIssuer(issuer string) *StatisticsQ {
	if q.Err != nil {
		return q
	}

	q.sql = q.sql.Where("stat.asset_issuer = ?", issuer)
	return q
}

// Select loads the results of the query specified by `q` into `dest`.
func (q *StatisticsQ) Select(dest interface{}) error {
	if q.Err != nil {
		return q.Err
	}

	q.Err = q.parent.Select(dest, q.sql)
	return q.Err
}

var selectStats = sq.Select("stat.*").From("statistics stat")
