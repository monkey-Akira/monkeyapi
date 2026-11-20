package operation_setting

import "errors"

// Amounts use the same raw quota units as consume logs and the existing form.
type CheckinSetting struct {
	Enabled                        bool `json:"enabled"`
	MinQuota                       int  `json:"min_quota"`
	MaxQuota                       int  `json:"max_quota"`
	MinPreviousDayRequests         int  `json:"min_previous_day_requests"`
	MinSingleRedemptionQuota       int  `json:"min_single_redemption_quota"`
	Last10PercentConsumeQuota      int  `json:"last_10_percent_consume_quota"`
	TwentyToTenPercentConsumeQuota int  `json:"twenty_to_ten_percent_consume_quota"`
}

const CheckinMinConsumption int64 = 10

func (s CheckinSetting) Validate() error {
	// The fixed low-consumption rewards must remain inside the configured range.
	if s.MinQuota < 1 || s.MinQuota > 5 || s.MaxQuota < 7 || s.MaxQuota > 2147483647 {
		return errors.New("签到最小额度须为1～5，最大额度须为7～2147483647；低消费奖励固定为5或5～7")
	}
	if s.MinPreviousDayRequests < 0 || s.MinPreviousDayRequests > 2147483647 ||
		s.MinSingleRedemptionQuota < 0 || s.MinSingleRedemptionQuota > 2147483647 {
		return errors.New("签到请求次数和兑换码门槛须为0～2147483647的整数")
	}
	topMin := (int64(s.MaxQuota)*9 + 9) / 10
	secondMin := (int64(s.MaxQuota)*8 + 9) / 10
	// Thresholds above 50 keep the two explicit low-consumption rules intact.
	if s.Last10PercentConsumeQuota <= 50 || s.Last10PercentConsumeQuota > 2147483647 ||
		int64(s.Last10PercentConsumeQuota)*15/100 < topMin {
		return errors.New("第一档门槛须大于50，且门槛的15%必须达到最大奖励的90%（向上取整）")
	}
	if s.TwentyToTenPercentConsumeQuota != 0 &&
		(s.TwentyToTenPercentConsumeQuota <= 50 ||
			s.TwentyToTenPercentConsumeQuota >= s.Last10PercentConsumeQuota ||
			int64(s.TwentyToTenPercentConsumeQuota)*15/100 < secondMin) {
		return errors.New("第二档填0关闭；启用时须大于50且低于第一档，门槛的15%必须达到最大奖励的80%（向上取整）")
	}
	return nil
}

// Reward uses an injected random draw so boundaries and the exact 1% lottery
// can be tested deterministically. Production passes math/rand.Intn.
func (s CheckinSetting) Reward(consumed int64, intn func(int) int) (int, error) {
	if err := s.Validate(); err != nil {
		return 0, err
	}
	if consumed < CheckinMinConsumption {
		return 0, errors.New("昨日消耗不足10额度，未达到签到要求")
	}
	if consumed < 20 {
		return 5, nil
	}
	if consumed <= 50 {
		return 5 + intn(3), nil
	}

	// Divide before multiplying to avoid overflowing large consume totals.
	capQuota := consumed/100*15 + consumed%100*15/100
	upper := int(min(int64(s.MaxQuota), capQuota))
	lower := s.MinQuota
	topMin := int((int64(s.MaxQuota)*9 + 9) / 10)
	secondMin := int((int64(s.MaxQuota)*8 + 9) / 10)

	switch {
	case consumed >= int64(s.Last10PercentConsumeQuota):
		lower = max(lower, topMin)
	case consumed >= int64(s.MaxQuota)*4 && intn(100) == 0:
		// The sole jackpot bypasses the ordinary 15% cap, never MaxQuota.
		lower, upper = max(lower, topMin), s.MaxQuota
	case s.TwentyToTenPercentConsumeQuota > 0 &&
		consumed >= int64(s.TwentyToTenPercentConsumeQuota):
		lower, upper = max(lower, secondMin), min(upper, topMin-1)
	case consumed < int64(s.MaxQuota)*2:
		upper = min(upper, int(int64(s.MaxQuota)*6/10))
	default:
		lowEnd := int(int64(upper)*6/10) - 1
		highStart := max(lower, lowEnd+1)
		highEnd := int((int64(upper)*9+9)/10) - 1
		switch {
		case lowEnd >= lower && highEnd >= highStart:
			if intn(100) < 85 {
				upper = lowEnd
			} else {
				lower, upper = highStart, highEnd
			}
		case lowEnd >= lower:
			upper = lowEnd
		case highEnd >= highStart:
			lower, upper = highStart, highEnd
		}
	}
	if upper < lower {
		return 0, errors.New("签到奖励配置的有效区间为空，请联系管理员")
	}
	if upper == lower {
		return lower, nil
	}
	return lower + intn(upper-lower+1), nil
}
