package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"math"
	"sort"
	"strconv"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/paymentauditlog"
	"github.com/Wei-Shaw/sub2api/ent/paymentorder"
)

// --- Dashboard & Analytics ---

func (s *PaymentService) GetDashboardStats(ctx context.Context, days int) (*DashboardStats, error) {
	if days <= 0 {
		days = 30
	}
	now := time.Now()
	since := now.AddDate(0, 0, -days)
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	trendStart := dashboardTrendStart(now, days)
	trendSince := trendStart.AddDate(0, 0, -29)

	paidStatuses := []string{OrderStatusCompleted, OrderStatusPaid, OrderStatusRecharging}

	earliestOrders, err := s.entClient.PaymentOrder.Query().
		Where(
			paymentorder.StatusIn(paidStatuses...),
			paymentorder.PaidAtNotNil(),
		).
		Order(paymentorder.ByPaidAt()).
		Limit(1).
		All(ctx)
	if err != nil {
		return nil, err
	}
	var earliestPaidAt *time.Time
	if len(earliestOrders) > 0 && earliestOrders[0].PaidAt != nil {
		earliestPaidAt = earliestOrders[0].PaidAt
	}

	orders, err := s.entClient.PaymentOrder.Query().
		Where(
			paymentorder.StatusIn(paidStatuses...),
			paymentorder.PaidAtGTE(trendSince),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}

	windowOrders := make([]*dbent.PaymentOrder, 0, len(orders))
	for _, order := range orders {
		if order.PaidAt != nil && !order.PaidAt.Before(since) {
			windowOrders = append(windowOrders, order)
		}
	}

	st := &DashboardStats{}
	computeBasicStats(st, windowOrders, todayStart)

	st.PendingOrders, err = s.entClient.PaymentOrder.Query().
		Where(paymentorder.StatusEQ(OrderStatusPending)).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	st.DailySeries = buildDailySeries(windowOrders, since, days)
	st.RevenueTrend = buildRevenueTrend(orders, trendStart, days, earliestPaidAt)
	st.PaymentMethods = buildMethodDistribution(windowOrders)
	st.TopUsers = buildTopUsers(windowOrders)

	return st, nil
}

func computeBasicStats(st *DashboardStats, orders []*dbent.PaymentOrder, todayStart time.Time) {
	st.TotalAmount = make(CurrencyAmounts)
	st.TodayAmount = make(CurrencyAmounts)
	st.AvgAmount = make(CurrencyAmounts)
	currencyCounts := make(map[string]int)
	var todayCount int
	for _, o := range orders {
		currency := PaymentOrderCurrency(o)
		st.TotalAmount[currency] += o.PayAmount
		currencyCounts[currency]++
		if o.PaidAt != nil && !o.PaidAt.Before(todayStart) {
			st.TodayAmount[currency] += o.PayAmount
			todayCount++
		}
	}
	st.TotalCount = len(orders)
	st.TodayCount = todayCount
	for currency, totalAmount := range st.TotalAmount {
		st.AvgAmount[currency] = roundAmount(totalAmount / float64(currencyCounts[currency]))
	}
	roundCurrencyAmounts(st.TotalAmount)
	roundCurrencyAmounts(st.TodayAmount)
}

func buildDailySeries(orders []*dbent.PaymentOrder, since time.Time, days int) []DailyStats {
	dailyMap := make(map[string]*DailyStats)
	for _, o := range orders {
		if o.PaidAt == nil {
			continue
		}
		date := o.PaidAt.Format("2006-01-02")
		ds, ok := dailyMap[date]
		if !ok {
			ds = &DailyStats{Date: date, Amount: make(CurrencyAmounts)}
			dailyMap[date] = ds
		}
		ds.Amount[PaymentOrderCurrency(o)] += o.PayAmount
		ds.Count++
	}
	series := make([]DailyStats, 0, days)
	for i := 0; i < days; i++ {
		date := since.AddDate(0, 0, i+1).Format("2006-01-02")
		if ds, ok := dailyMap[date]; ok {
			roundCurrencyAmounts(ds.Amount)
			series = append(series, *ds)
		} else {
			series = append(series, DailyStats{Date: date, Amount: make(CurrencyAmounts)})
		}
	}
	return series
}

func dashboardTrendStart(now time.Time, days int) time.Time {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return today.AddDate(0, 0, -days)
}

func buildRevenueTrend(orders []*dbent.PaymentOrder, start time.Time, days int, earliestPaidAt *time.Time) []RevenueTrendPoint {
	dailyAmounts := make(map[string]CurrencyAmounts)
	currencies := make(map[string]struct{})
	for _, order := range orders {
		if order.PaidAt == nil {
			continue
		}

		paidAt := order.PaidAt.In(start.Location())
		paidDate := time.Date(paidAt.Year(), paidAt.Month(), paidAt.Day(), 0, 0, 0, 0, start.Location())
		dateKey := paidDate.Format("2006-01-02")
		currency := PaymentOrderCurrency(order)
		currencies[currency] = struct{}{}
		if _, ok := dailyAmounts[dateKey]; !ok {
			dailyAmounts[dateKey] = make(CurrencyAmounts)
		}
		dailyAmounts[dateKey][currency] += order.PayAmount
	}

	series := make([]RevenueTrendPoint, 0, days)
	for i := 0; i < days; i++ {
		date := start.AddDate(0, 0, i)
		point := RevenueTrendPoint{
			Date:             date.Format("2006-01-02"),
			Rolling7dAmount:  make(NullableCurrencyAmounts),
			Rolling30dAmount: make(NullableCurrencyAmounts),
		}
		for currency := range currencies {
			point.Rolling7dAmount[currency] = rollingRevenueAmount(dailyAmounts, date, earliestPaidAt, currency, 7)
			point.Rolling30dAmount[currency] = rollingRevenueAmount(dailyAmounts, date, earliestPaidAt, currency, 30)
		}
		series = append(series, point)
	}
	return series
}

func rollingRevenueAmount(dailyAmounts map[string]CurrencyAmounts, date time.Time, earliestPaidAt *time.Time, currency string, windowDays int) *float64 {
	if earliestPaidAt == nil {
		return nil
	}
	windowStart := date.AddDate(0, 0, 1-windowDays)
	earliestLocal := earliestPaidAt.In(date.Location())
	earliestDate := time.Date(earliestLocal.Year(), earliestLocal.Month(), earliestLocal.Day(), 0, 0, 0, 0, date.Location())
	if windowStart.Before(earliestDate) {
		return nil
	}

	var amount float64
	for dayOffset := 0; dayOffset < windowDays; dayOffset++ {
		day := date.AddDate(0, 0, -dayOffset).Format("2006-01-02")
		amount += dailyAmounts[day][currency]
	}
	amount = roundAmount(amount)
	return &amount
}

func buildMethodDistribution(orders []*dbent.PaymentOrder) []PaymentMethodStat {
	methodMap := make(map[string]*PaymentMethodStat)
	for _, o := range orders {
		ms, ok := methodMap[o.PaymentType]
		if !ok {
			ms = &PaymentMethodStat{Type: o.PaymentType, Amount: make(CurrencyAmounts)}
			methodMap[o.PaymentType] = ms
		}
		ms.Amount[PaymentOrderCurrency(o)] += o.PayAmount
		ms.Count++
	}
	methods := make([]PaymentMethodStat, 0, len(methodMap))
	for _, ms := range methodMap {
		roundCurrencyAmounts(ms.Amount)
		methods = append(methods, *ms)
	}
	sort.Slice(methods, func(i, j int) bool {
		return methods[i].Type < methods[j].Type
	})
	return methods
}

func buildTopUsers(orders []*dbent.PaymentOrder) TopUsersByCurrency {
	userMap := make(map[string]map[int64]*TopUserStat)
	for _, o := range orders {
		currency := PaymentOrderCurrency(o)
		users, ok := userMap[currency]
		if !ok {
			users = make(map[int64]*TopUserStat)
			userMap[currency] = users
		}
		us, ok := users[o.UserID]
		if !ok {
			us = &TopUserStat{UserID: o.UserID, Email: o.UserEmail}
			users[o.UserID] = us
		}
		us.Amount += o.PayAmount
	}
	result := make(TopUsersByCurrency, len(userMap))
	for currency, users := range userMap {
		userList := make([]*TopUserStat, 0, len(users))
		for _, us := range users {
			us.Amount = roundAmount(us.Amount)
			userList = append(userList, us)
		}
		sort.Slice(userList, func(i, j int) bool {
			return userList[i].Amount > userList[j].Amount
		})
		limit := topUsersLimit
		if len(userList) < limit {
			limit = len(userList)
		}
		result[currency] = make([]TopUserStat, 0, limit)
		for i := 0; i < limit; i++ {
			result[currency] = append(result[currency], *userList[i])
		}
	}
	return result
}

func roundCurrencyAmounts(amounts CurrencyAmounts) {
	for currency, amount := range amounts {
		amounts[currency] = roundAmount(amount)
	}
}

func roundAmount(amount float64) float64 {
	return math.Round(amount*100) / 100
}

// --- Audit Logs ---

func (s *PaymentService) writeAuditLog(ctx context.Context, oid int64, action, op string, detail map[string]any) {
	dj, _ := json.Marshal(detail)
	_, err := s.entClient.PaymentAuditLog.Create().SetOrderID(strconv.FormatInt(oid, 10)).SetAction(action).SetDetail(string(dj)).SetOperator(op).Save(ctx)
	if err != nil {
		slog.Error("audit log failed", "orderID", oid, "action", action, "error", err)
	}
}

func (s *PaymentService) GetOrderAuditLogs(ctx context.Context, oid int64) ([]*dbent.PaymentAuditLog, error) {
	return s.entClient.PaymentAuditLog.Query().Where(paymentauditlog.OrderIDEQ(strconv.FormatInt(oid, 10))).Order(paymentauditlog.ByCreatedAt()).All(ctx)
}
