package statistics

import (
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/redis"
	"time"
	"errors"
)

type ManagerInterface interface {
	// Gets statistics for account-asset pair, updates it with opAmount and returns stats
	UpdateGet(account string, assetCode string, counterparty xdr.AccountType,
		isIncome bool, now time.Time, txHash string, opIndex int, opAmount int64) (result map[xdr.AccountType]history.AccountStatistics, err error)

	// Cancels Op - removes it from processed ops and subtracts from stats
	CancelOp(account, assetCode string, counterparty xdr.AccountType, isIncome bool, now time.Time,
		txHash string, opIndex int, opAmount int64) error
}

type Manager struct {
	historyQ             history.QInterface
	counterparties       []xdr.AccountType
	statisticsTimeOut    time.Duration
	processedOpTimeOut   time.Duration
	numOfRetires         int
	connectionProvider   redis.ConnectionProviderInterface
	processedOpProvider  redis.ProcessedOpProviderInterface
	accountStatsProvider redis.AccountStatisticsProviderInterface
	log                  *log.Entry
}

// Creates new statistics manager. counterparties MUST BE FULL ARRAY OF COUTERPARTIES.
// statisticsTimeOut must be bigger then ledger's close time (~0.5 hour is recommended)
// timeout for statistics must be greater then for processed op
func NewManager(historyQ history.QInterface, counterparties []xdr.AccountType, statisticsTimeOut, processedOpTimeOut time.Duration) *Manager {
	return &Manager{
		historyQ:           historyQ,
		counterparties:     counterparties,
		statisticsTimeOut:  statisticsTimeOut,
		processedOpTimeOut: processedOpTimeOut,
		numOfRetires:       5,
		log:                log.WithField("service", "statistics_manager"),
	}
}

func (m *Manager) getConnectionProvider() redis.ConnectionProviderInterface {
	if m.connectionProvider == nil {
		m.connectionProvider = redis.NewConnectionProvider()
	}
	return m.connectionProvider
}

func (m *Manager) getProcessedOpProvider(conn redis.ConnectionInterface) redis.ProcessedOpProviderInterface {
	if m.processedOpProvider == nil {
		m.processedOpProvider = redis.NewProcessedOpProvider(conn)
	}
	return m.processedOpProvider
}

func (m *Manager) getAccountStatsProvider(conn redis.ConnectionInterface) redis.AccountStatisticsProviderInterface {
	if m.accountStatsProvider == nil {
		m.accountStatsProvider = redis.NewAccountStatisticsProvider(conn)
	}
	return m.accountStatsProvider
}

func (m *Manager) CancelOp(account, assetCode string, counterparty xdr.AccountType, isIncome bool, now time.Time,
txHash string, opIndex int, opAmount int64) error {
	for i := 0; i < m.numOfRetires; i++ {
		m.log.WithField("retry", i).Debug("CancelOp started new retry")
		var needRetry bool
		needRetry, err := m.cancelOp(account, assetCode, counterparty, isIncome, now, txHash, opIndex, opAmount)
		if err != nil {
			return err
		}

		if !needRetry {
			return nil
		}
	}

	return errors.New("Failed to cancel op")
}

// Returns true if retry needed
func (m *Manager) cancelOp(account, assetCode string, counterparty xdr.AccountType, isIncome bool, now time.Time,
txHash string, opIndex int, opAmount int64) (bool, error) {
	m.log.Debug("Getting new connection")
	conn := m.getConnectionProvider().GetConnection()
	defer conn.Close()

	// Check if op is still in redis
	processedOp, err := m.getProcessedOp(txHash, opIndex, conn)
	if err != nil {
		return false, err
	}

	if processedOp == nil {
		// op is already canceled - remove op watch
		m.log.Debug("Op is canceled - unwatching")
		err = conn.UnWatch()
		return false, err
	}

	// Get stats
	accountStats, err := m.getAccountStatistics(account, assetCode, conn)
	if err != nil {
		m.log.WithError(err).Error("Failed to get account statistics")
		return false, err
	}

	if accountStats == nil {
		// no need to cancel
		m.log.Debug("Stats are not in redis - no need to cancel operation")
		err = conn.UnWatch()
		return false, err
	}

	for key, value := range accountStats.AccountsStatistics {
		value.ClearObsoleteStats(now)
		if key == counterparty {
			value.Update(-opAmount, processedOp.TimeUpdated, now, isIncome)
		}
		accountStats.AccountsStatistics[key] = value
	}

	// Update stats and del op processed
	// 4 Start multi
	m.log.Debug("Starting multi")
	err = conn.Multi()
	if err != nil {
		return false, err
	}

	// 5. Save to redis stats
	err = m.getAccountStatsProvider(conn).Insert(accountStats, m.statisticsTimeOut)
	if err != nil {
		return false, err
	}

	processedOp = redis.NewProcessedOp(txHash, opIndex, opAmount, now)
	// 6. Mark Op processed
	err = m.getProcessedOpProvider(conn).Delete(txHash, opIndex)
	if err != nil {
		return false, err
	}

	// commit
	isOk, err := conn.Exec()
	if err != nil {
		return false, err
	}

	return !isOk, nil


}

func (m *Manager) UpdateGet(account string, assetCode string, counterparty xdr.AccountType,
	isIncome bool, now time.Time, txHash string, opIndex int, opAmount int64) (result map[xdr.AccountType]history.AccountStatistics, err error) {
	var accountStats *redis.AccountStatistics
	for i := 0; i < m.numOfRetires; i++ {
		m.log.WithField("retry", i).Debug("UpdateGet started new retry")
		var needRetry bool
		accountStats, needRetry, err = m.updateGet(account, assetCode, counterparty, isIncome, now, txHash, opIndex, opAmount)
		if err != nil {
			return nil, err
		}

		if !needRetry {
			return accountStats.AccountsStatistics, nil
		}
	}

	return nil, errors.New("Failed to Update and Get Account stats")
}

func (m *Manager) updateGet(account string, assetCode string, counterparty xdr.AccountType,
	isIncome bool, now time.Time, txHash string, opIndex int, opAmount int64) (*redis.AccountStatistics, bool, error) {
	m.log.Debug("Getting new connection")
	conn := m.getConnectionProvider().GetConnection()
	defer conn.Close()

	// 1. Check if op processed
	processedOp, err := m.getProcessedOp(txHash, opIndex, conn)
	if err != nil {
		return nil, false, err
	}

	if processedOp != nil {
		// remove op watch
		m.log.Debug("Op is processed - unwatching")
		err := conn.UnWatch()
		if err != nil {
			return nil, false, err
		}

		return m.manageProcessedOp(conn, account, assetCode, now)
	}


	accountStats, err := m.getAccountStatistics(account, assetCode, conn)
	if err != nil {
		return nil, false, err
	}

	if accountStats == nil {
		// try get from db
		m.log.Debug("Getting stats from histroy")
		accountStats, err = m.tryGetStatisticsFromHistory(account, assetCode, now)
		if err != nil {
			m.log.WithError(err).Error("Failed to get stats from history")
			return nil, false, err
		}
	}
	// 4. Update stats and set op processed
	m.updateStats(accountStats, counterparty, isIncome, opAmount, now)
	// 4.1 Start multi
	m.log.Debug("Starting multi")
	err = conn.Multi()
	if err != nil {
		return nil, false, err
	}

	// 5. Save to redis stats
	err = m.getAccountStatsProvider(conn).Insert(accountStats, m.statisticsTimeOut)
	if err != nil {
		return nil, false, err
	}

	processedOp = redis.NewProcessedOp(txHash, opIndex, opAmount, now)
	// 6. Mark Op processed
	err = m.getProcessedOpProvider(conn).Insert(processedOp, m.processedOpTimeOut)
	if err != nil {
		return nil, false, err
	}

	// commit
	isOk, err := conn.Exec()
	if err != nil {
		return nil, false, err
	}

	if !isOk {
		return nil, true, nil
	}
	return accountStats, false, nil
}

func (m *Manager) getProcessedOp(txHash string, opIndex int, conn redis.ConnectionInterface) (*redis.ProcessedOp, error) {
	// 1. Watch op
	m.log.Debug("Setting watch for processed op key")
	opKey := redis.GetProcessedOpKey(txHash, opIndex)
	err := conn.Watch(opKey)
	if err != nil {
		return nil, err
	}

	// 2. Get op
	m.log.Debug("Checking if op was processed")
	processedOpProvider := m.getProcessedOpProvider(conn)
	processedOp, err := processedOpProvider.Get(txHash, opIndex)
	if err != nil {
		m.log.WithError(err).Error("Failed to get processed op")
		return nil, err
	}

	return processedOp, nil
}

func (m *Manager) getAccountStatistics(account, assetCode string, conn redis.ConnectionInterface) (*redis.AccountStatistics, error) {
	// 1. Watch stats
	m.log.Debug("Watching account stats")
	statsKey := redis.GetAccountStatisticsKey(account, assetCode)
	err := conn.Watch(statsKey)
	if err != nil {
		m.log.WithError(err).Error("Failed to watch stats key")
		return nil, err
	}
	// 2. Get stats
	m.log.Debug("Getting account stats from redis")
	accountStatsProvider := m.getAccountStatsProvider(conn)
	accountStats, err := accountStatsProvider.Get(account, assetCode, m.counterparties)
	if err != nil {
		m.log.WithError(err).Error("Failed to get stats from redis")
		return nil, err
	}

	return accountStats, nil
}

func (m *Manager) manageProcessedOp(conn redis.ConnectionInterface, account string, assetCode string, now time.Time) (*redis.AccountStatistics, bool, error) {
	// try get stats from redis
	accountStatsProvider := m.getAccountStatsProvider(conn)
	accountStats, err := accountStatsProvider.Get(account, assetCode, m.counterparties)
	if err != nil {
		return nil, false, err
	}

	if accountStats == nil {
		// try get stats from history
		accountStats, err = m.tryGetStatisticsFromHistory(account, assetCode, now)
		if err != nil {
			return nil, false, err
		}

		err := accountStatsProvider.Insert(accountStats, m.statisticsTimeOut)
		if err != nil {
			return nil, false, err
		}
		return accountStats, false, err
	}
	return accountStats, false, nil
}

func (m *Manager) updateStats(accountStats *redis.AccountStatistics, counterparty xdr.AccountType, isIncome bool, opAmount int64, now time.Time) {
	_, ok := accountStats.AccountsStatistics[counterparty]
	if !ok {
		accountStats.AccountsStatistics[counterparty] = history.NewAccountStatistics(accountStats.Account, accountStats.AssetCode, counterparty)
	}
	for key, value := range accountStats.AccountsStatistics {
		value.ClearObsoleteStats(now)
		if key == counterparty {
			value.Update(opAmount, now, now, isIncome)
		}
		accountStats.AccountsStatistics[key] = value
	}
}

func (m *Manager) tryGetStatisticsFromHistory(account, assetCode string, now time.Time) (*redis.AccountStatistics, error) {
	accountStats := redis.NewAccountStatistics(account, assetCode, make(map[xdr.AccountType]history.AccountStatistics))
	err := m.historyQ.GetStatisticsByAccountAndAsset(accountStats.AccountsStatistics, account, assetCode, now)
	if err != nil {
		return nil, err
	}
	return accountStats, nil
}
