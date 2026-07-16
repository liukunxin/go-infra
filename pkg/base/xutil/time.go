package xutil

import "time"

// CurrentDayLeftTime 获取当天剩余时间
func CurrentDayLeftTime() time.Duration {
	todayLast := time.Now().Format("2006-01-02") + " 23:59:59"
	todayLastTime, _ := time.ParseInLocation("2006-01-02 15:04:05", todayLast, time.Local)
	remainSecond := time.Duration(todayLastTime.Unix()-time.Now().Local().Unix()) * time.Second
	return remainSecond
}

// CurrentWeekLeftTime 获取当周剩余时间
func CurrentWeekLeftTime() time.Duration {
	now := time.Now()
	offset := int(time.Monday - now.Weekday())
	//周日做特殊判断 因为time.Monday = 0
	if offset > 0 {
		offset = -6
	}

	year, month, day := now.Date()
	thisWeek := time.Date(year, month, day, 0, 0, 0, 0, time.Local)
	endTime := thisWeek.AddDate(0, 0, offset+6+7*0)
	return endTime.Sub(now)
}

// CurrentMonthLeftTime 获取当月剩余时间
func CurrentMonthLeftTime() time.Duration {
	now := time.Now()
	year, month, _ := now.Date()
	thisMonth := time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
	endTime := thisMonth.AddDate(0, 0+1, -1).Add(24 * time.Hour).Add(-1 * time.Second)
	return endTime.Sub(now)
}

// WeekIntervalTime 获取某周的开始和结束时间,week为0本周,-1上周，1下周以此类推
func WeekIntervalTime(week int) (startTime, endTime time.Time) {
	now := time.Now()
	offset := int(time.Monday - now.Weekday())
	//周日做特殊判断 因为time.Monday = 0
	if offset > 0 {
		offset = -6
	}

	year, month, day := now.Date()
	thisWeek := time.Date(year, month, day, 0, 0, 0, 0, time.Local)
	startTime = thisWeek.AddDate(0, 0, offset+7*week)
	endTime = thisWeek.AddDate(0, 0, offset+6+7*week)

	return startTime, endTime
}

// MonthIntervalTime 获取某月的开始和结束时间mon为0本月,-1上月，1下月以此类推
func MonthIntervalTime(mon int) (startTime, endTime time.Time) {
	year, month, _ := time.Now().Date()
	thisMonth := time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
	startTime = thisMonth.AddDate(0, mon, 0)
	endTime = thisMonth.AddDate(0, mon+1, -1).Add(24 * time.Hour).Add(-1 * time.Second)
	return startTime, endTime
}
