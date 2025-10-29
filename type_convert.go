package gobatis

import (
	"strconv"
	"time"
)

// type convert tools function
// contains: int64, byte, rune, float64, bool, string

// AsInt convert string to int64
func AsInt(v string) (int64, error) {
	return strconv.ParseInt(v, 10, 64)
}

// AsByte convert string to byte
func AsByte(v string) (byte, error) {
	if len(v) == 0 {
		return 0x00, nil
	}
	return v[0], nil
}

// AsRune convert string to rune
func AsRune(v string) (rune, error) {
	if len(v) == 0 {
		return 0x00, nil
	}
	return []rune(v)[0], nil
}

// AsFloat convert string to float64
func AsFloat(v string) (float64, error) {
	return strconv.ParseFloat(v, 64)
}

// AsBool convert string to bool
func AsBool(v string) (bool, error) {
	return strconv.ParseBool(v)
}

// AsString convert string to string
func AsString(v string) string {
	return v
}

type DateTime time.Time

// NewDateTimeFromTime create DateTime from time.Time
func NewDateTimeFromTime(t time.Time) DateTime {
	return DateTime(t)
}

// NewDateTimeFromNow create DateTime from now
func NewDateTimeFromNow() DateTime {
	return NewDateTimeFromTime(time.Now())
}

// StartOfMonth return start of month
func (dt DateTime) StartOfMonth() DateTime {
	t := time.Time(dt)
	return DateTime(time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location()))
}

// EndOfMonth return end of month
func (dt DateTime) EndOfMonth() DateTime {
	t := time.Time(dt.StartOfMonth())
	return DateTime(
		time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, t.Location()).
			AddDate(0, 0, -1))
}

// AddYear add year
func (dt DateTime) AddYear(y int) DateTime {
	t := time.Time(dt)
	return DateTime(t.AddDate(y, 0, 0))
}

// AddMonth add month
func (dt DateTime) AddMonth(m int) DateTime {
	t := time.Time(dt)
	return DateTime(t.AddDate(0, m, 0))
}

// AddDay add day
func (dt DateTime) AddDay(d int) DateTime {
	t := time.Time(dt)
	return DateTime(t.AddDate(0, 0, d))
}

// AddDuration add duration
func (dt DateTime) AddDuration(duration time.Duration) DateTime {
	t := time.Time(dt)
	return DateTime(t.Add(duration))
}

// AddDurationInText add duration in text
func (dt DateTime) AddDurationInText(duration string) (DateTime, error) {
	ndt, err := time.ParseDuration(duration)
	if err != nil {
		return DateTime{}, err
	}
	t := time.Time(dt)
	return DateTime(t).AddDuration(ndt), nil
}

// StartOfWeek return start of week (Monday)
func (dt DateTime) StartOfWeek() DateTime {
	t := time.Time(dt)
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	daysToSubtract := weekday - 1
	return DateTime(t).AddDay(-daysToSubtract)
}

// EndOfWeek return end of week (Sunday)
func (dt DateTime) EndOfWeek() DateTime {
	t := time.Time(dt)
	weekday := int(t.Weekday())
	daysToAdd := 7 - weekday
	return DateTime(t).AddDay(daysToAdd)
}

// AddWeek add week
func (dt DateTime) AddWeek(week int) DateTime {
	t := time.Time(dt)
	return DateTime(t.AddDate(0, 0, week*7))
}

// StartOfDay return start of day
func (dt DateTime) StartOfDay() DateTime {
	t := time.Time(dt)
	return DateTime(time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()))
}

// EndOfDay return end of day
// some database engine like mysql, the precison of datetime is microseconds, so we omit the nsec parameters.
// if you want to set the nsec, you can use DateTime.AddDurationInText() or DateTime.AddDuration() to set the nsec.
func (dt DateTime) EndOfDay() DateTime {
	t := time.Time(dt)
	return DateTime(time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, t.Location()))
}

// AddHour add hour
func (dt DateTime) AddHour(h int) DateTime {
	return dt.AddDuration(time.Duration(h) * time.Hour)
}

// AddMinute add minute
func (dt DateTime) AddMinute(m int) DateTime {
	return dt.AddDuration(time.Duration(m) * time.Minute)
}

// AddSecond add second
func (dt DateTime) AddSecond(s int) DateTime {
	return dt.AddDuration(time.Duration(s) * time.Second)
}

// Format time with the given format string and return the formatted string
func (dt DateTime) Format(format TimeFormat) string {
	return time.Time(dt).Format(string(format))
}

// AsDate convert string to time.Time
func AsDate(v string, format string) (DateTime, error) {
	tq, err := time.Parse(format, v)
	return NewDateTimeFromTime(tq), err
}

// AsLocation convert string to *time.Location
func AsLocation(v string) (*time.Location, error) {
	return time.LoadLocation(v)
}

type TimeFormat string

const (
	TimeFormatYmd             TimeFormat = "2006-1-2"
	TimeFormatYmdWithSlash    TimeFormat = "2006/1/2"
	TimeFormatYmdInCN         TimeFormat = "2006年1月2日"
	TimeFormatYmdHms          TimeFormat = "2006-1-2 15:04:05"
	TimeFormatYmdHmsWithSlash TimeFormat = "2006/1/2 15:04:05"
	TimeFormatYmdHmsInCN      TimeFormat = "2006年1月2日 15点04分05"
	TimeFormatHms             TimeFormat = "15:04:05"
	TimeFormatHmsInCN         TimeFormat = "15点04分05"
)

// AsTime convert string to time.Time
func (tf TimeFormat) AsTime(v string) (DateTime, error) {
	tq, err := time.Parse(string(tf), v)
	return DateTime(tq), err
}

// AsTimeInLocation convert string to time.Time
func (tf TimeFormat) AsTimeInLocation(v string, loc *time.Location) (DateTime, error) {
	tq, err := time.ParseInLocation(string(tf), v, loc)
	return DateTime(tq), err
}

// AsTimeInLocationName convert string to time.Time
func (tf TimeFormat) AsTimeInLocationName(v string, locationName string) (DateTime, error) {
	loc, err := AsLocation(locationName)
	if err != nil {
		return DateTime{}, err
	}
	tq, err := time.ParseInLocation(string(tf), v, loc)
	return DateTime(tq), err
}
