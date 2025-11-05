package gobatis

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// AsInt type convert tools function
// contains: int64, byte, rune, float64, bool, string
// time.Time
// simple example:
//
// Parse string into int64
// v, _ := AsInt("10")
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

// deprecated
// AsString convert interface{} to string
// Not recommend to use AsString.
func AsString(v interface{}) (string, error) {
	switch v := v.(type) {
	case int:
		return strconv.Itoa(v), nil
	case int8:
		return strconv.FormatInt(int64(v), 10), nil
	case int16:
		return strconv.FormatInt(int64(v), 10), nil
	case int32:
		return strconv.FormatInt(int64(v), 10), nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	case uint:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint8:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint16:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint32:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint64:
		return strconv.FormatUint(v, 10), nil
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32), nil
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	case time.Time:
		return v.Format(TimeFormatYmdHis.String()), nil
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	case []rune:
		return string(v), nil
	case *time.Time:
		if v != nil {
			return v.Format(TimeFormatYmdHis.String()), nil
		}
		return ``, fmt.Errorf(`gobatis: value is nil`)
	case DateTime:
		return v.Format(TimeFormatYmdHis), nil
	case *DateTime:
		if v != nil {
			return v.Format(TimeFormatYmdHis), nil
		}
		return ``, fmt.Errorf(`gobatis: value is nil`)
	default:
		return fmt.Sprintf(`%v`, v), nil
	}
}

// DateTime type convert tools function
// contains: int64, byte, rune, float64, bool, string
// time.Time
// simple example:
//
// Parse string into Time
// 3. tq, _ := AsDateTime("2021-01-01", TimeFormat(`Y-m-d`).AsTimeFormat().String())
//	  println(tq.Format(TimeFormatYmdInCN))
//
// or you can use AsTime() function
//	dt, _ := TimeFormat(`Y-m-d`).AsTimeFormat().AsTime("2021-01-01")
//	println(dt.Format(TimeFormatYmdInCN)
//
// also, you can simplely use like this:
//  dt, _ := TimeFormatYmd.AsTime("2021-01-01")
//  println(dt.Format(TimeFormatYmdInCN)
type DateTime time.Time

// NewDateTimeFromTime create DateTime from time.Time
func NewDateTimeFromTime(t time.Time) DateTime {
	return DateTime(t)
}

// NewDateTimeFromNow create DateTime from now
func NewDateTimeFromNow() DateTime {
	return NewDateTimeFromTime(time.Now())
}

// ToTime convert DateTime to time.Time
func (dt DateTime) ToTime() time.Time {
	return time.Time(dt)
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
// some database engine like mysql, the precison of datetime is microseconds, so we omit the nsec part.
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

// Equal compare two DateTime
func (dt DateTime) Equal(t DateTime) bool {
	return time.Time(dt).Equal(time.Time(t))
}

// EqualInTime compare DateTime with time.Time is Equal
func (dt DateTime) EqualInTime(t time.Time) bool {
	return time.Time(dt).Equal(t)
}

// BeforeOrEqual compare DateTime with another DateTime
func (dt DateTime) BeforeOrEqual(t DateTime) bool {
	return dt.Before(t) || dt.Equal(t)
}

// Unix return unix timestamp
func (dt DateTime) Unix() int64 {
	return time.Time(dt).Unix()
}

// UnixNano return unix timestamp with nanosecond
func (dt DateTime) UnixNano() int64 {
	return time.Time(dt).UnixNano()
}

// Year return year
func (dt DateTime) Year() int {
	return time.Time(dt).Year()
}

// Month return month
func (dt DateTime) Month() time.Month {
	return time.Time(dt).Month()
}

// Day return day
func (dt DateTime) Day() int {
	return time.Time(dt).Day()
}

// Hour return hour
func (dt DateTime) Hour() int {
	return time.Time(dt).Hour()
}

// Minute return minute
func (dt DateTime) Minute() int {
	return time.Time(dt).Minute()
}

// Second return second
func (dt DateTime) Second() int {
	return time.Time(dt).Second()
}

// Nanosecond return nanosecond
func (dt DateTime) Nanosecond() int {
	return time.Time(dt).Nanosecond()
}

// IsZero return true if the DateTime is zero
func (dt DateTime) IsZero() bool {
	return time.Time(dt).IsZero()
}

// Before return true if the DateTime is before another DateTime
func (dt DateTime) Before(t DateTime) bool {
	return time.Time(dt).Before(time.Time(t))
}

// BeforeInTime return true if the DateTime is before another time.Time
func (dt DateTime) BeforeInTime(t time.Time) bool {
	return time.Time(dt).Before(t)
}

// After return true if the DateTime is after another DateTime
func (dt DateTime) After(t DateTime) bool {
	return time.Time(dt).After(time.Time(t))
}

// AfterInTime return true if the DateTime is after another time.Time
func (dt DateTime) AfterInTime(t time.Time) bool {
	return time.Time(dt).After(t)
}

// AsDateTime type convert tools function
// contains: int64, byte, rune, float64, bool, string
// time.Time
// simple example:
//
// Parse string into Time
// 3. tq, _ := AsDateTime("2021-01-01", TimeFormat(`Y-m-d`).AsTimeFormat().String())
//	  println(tq.Format(TimeFormatYmdInCN))
//
// or you can use AsTime() function
//	dt, _ := TimeFormat(`Y-m-d`).AsTimeFormat().AsTime("2021-01-01")
//	println(dt.Format(TimeFormatYmdInCN)
//
// also, you can simplely use like this:
//  dt, _ := TimeFormatYmd.AsTime("2021-01-01")
//  println(dt.Format(TimeFormatYmdInCN)
// AsDateTime convert string to DateTime
func AsDateTime(v string, format string) (DateTime, error) {
	tq, err := time.Parse(format, v)
	return NewDateTimeFromTime(tq), err
}

// AsLocation convert string to *time.Location
func AsLocation(v string) (*time.Location, error) {
	return time.LoadLocation(v)
}

// TimeFormat type convert tools function
// simple example:
//
// Parse string into Time
//
// 3. tq, _ := AsDateTime("2021-01-01", TimeFormat(`Y-m-d`).AsTimeFormat().String())
//	  println(tq.Format(TimeFormatYmdInCN))
//
// or you can use AsTime() function
//
//	dt, _ := TimeFormat(`Y-m-d`).AsTimeFormat().AsTime("2021-01-01")
//	println(dt.Format(TimeFormatYmdInCN)
//
// also, you can simplely use like this:
//
//  dt, _ := TimeFormatYmd.AsTime("2021-01-01")
//  println(dt.Format(TimeFormatYmdInCN)
type TimeFormat string

const (
	TimeFormatY           TimeFormat = "2006"
	TimeFormatYInCN       TimeFormat = "2006年"
	TimeFormatYm          TimeFormat = "2006-1"
	TimeFormatMy          TimeFormat = "1-2006"
	TimeFormatYmInCN      TimeFormat = "2006年1月"
	TimeFormatMyInCN      TimeFormat = "1月2006年"
	TimeFormatYmWithSlash TimeFormat = "2006/1"
	TimeFormatMyWithSlash TimeFormat = "1/2006"

	TimeFormatYmd                  TimeFormat = "2006-1-2"
	TimeFormatYmdWithSlash         TimeFormat = "2006/1/2"
	TimeFormatYmdInCN              TimeFormat = "2006年1月2日"
	TimeFormatYmdHms               TimeFormat = "2006-1-2 15:04:05"
	TimeFormatYmdHmsWithSlash      TimeFormat = "2006/1/2 15:04:05"
	TimeFormatYmdHmsInCN           TimeFormat = "2006年1月2日 15点04分05"
	TimeFormatYmdHmsWithSecondInCN TimeFormat = "2006年1月2日 15点04分05秒"
	TimeFormatHms                  TimeFormat = "15:04:05"
	TimeFormatHmsInCN              TimeFormat = "15点04分05"
	TimeFormatHmsWithSecondInCN    TimeFormat = "15点04分05秒"

	TimeFormatDmyHms TimeFormat = "2/1/2006 15:04:05"
	TimeFormatDmyHis TimeFormat = TimeFormatDmyHms

	TimeFormatMdy TimeFormat = "1-2-2006"

	TimeFormatYmdHis               TimeFormat = TimeFormatYmdHms
	TimeFormatYmdHisWithSlash      TimeFormat = TimeFormatYmdHmsWithSlash
	TimeFormatYmdHisInCN           TimeFormat = TimeFormatYmdHmsInCN
	TimeFormatHis                  TimeFormat = TimeFormatHms
	TimeFormatHisInCN              TimeFormat = TimeFormatHmsInCN
	TimeFormatYmdHisWithSecondInCN TimeFormat = TimeFormatYmdHmsWithSecondInCN
	TimeFormatHisWithSecondInCN    TimeFormat = TimeFormatHmsWithSecondInCN

	TimeFormatRFC3339        TimeFormat = time.RFC3339
	TimeFormatRFC3339Nano    TimeFormat = time.RFC3339Nano
	TimeFormatRFC822         TimeFormat = time.RFC822
	TimeFormatRFC850         TimeFormat = time.RFC850
	TimeFormatRFC822Z        TimeFormat = time.RFC822Z
	TimeFormatRFC1123        TimeFormat = time.RFC1123
	TimeFormatRFC1123Z       TimeFormat = time.RFC1123Z
	TimeFormatUnixDate       TimeFormat = time.UnixDate
	TimeFormatANSIC          TimeFormat = time.ANSIC
	TimeFormatRubyDate       TimeFormat = time.RubyDate
	TimeFormatDateTimeLayout TimeFormat = time.Layout
	TimeFormatKitchen        TimeFormat = time.Kitchen
	TimeFormatStamp          TimeFormat = time.Stamp
	TimeFormatStampMilli     TimeFormat = time.StampMilli
	TimeFormatStampMicro     TimeFormat = time.StampMicro
	TimeFormatStampNano      TimeFormat = time.StampNano
)

var replacerYmdHis = strings.NewReplacer(
	"Y", "2006",
	"y", "06",
	"a", "pm",
	"A", "PM",
	"m", "1",
	"d", "2",
	"H", "15",
	"h", "03",
	"g", "3",
	"i", "04",
	"s", "05",
)

// String return TimeFormat
func (tf TimeFormat) String() string {
	return string(tf)
}

// AsTimeFormat convert TimeFormat to time.Time format
func (tf TimeFormat) AsTimeFormat() TimeFormat {
	return TimeFormat(replacerYmdHis.Replace(string(tf)))
}

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

// columnName fetch name from struct field
// first from struct tag list found, if not, use it's field name to lower form.
func columnName(f reflect.StructField) string {
	for _, tag := range tagList {
		if n, ok := f.Tag.Lookup(tag); ok {
			for _, piece := range strings.Split(n, `;`) {
				if !strings.HasPrefix(piece, `column`) {
					continue
				}
				return piece[strings.Index(piece, `:`)+1:]
			}

			if strings.Contains(n, `,`) {
				return n[0:strings.Index(n, `,`)]
			}

			return n
		}
	}
	return f.Name
}

// AsMap convert struct to map, remain field name, if not define the following tag:
// `json`, `sql`, `db`, `expr`, `leopard`, `gorm`
// only support input param was struct & map, others panic
func AsMap(v any) Args {
	if v == nil {
		return nil
	}

	t := reflect.TypeOf(v)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	value := reflect.ValueOf(v)
	for value.Kind() == reflect.Ptr {
		value = value.Elem()
	}

	switch t.Kind() {
	case reflect.Struct:
		var data Args
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			fieldValue := value.Field(i)
			if field.Anonymous {
			dealAnonymous:
				// deal with anonymous field
				anonymousFieldType := field.Type
				anonymousFieldValue := fieldValue
				for j := 0; j < anonymousFieldType.NumField(); j++ {
					if anonymousFieldType.Field(j).Anonymous {
						field = anonymousFieldType.Field(j)
						fieldValue = anonymousFieldValue.Field(j)
						// embed field goto next for loop
						goto dealAnonymous
					}
					field := anonymousFieldType.Field(j)
					fieldValue := anonymousFieldValue.Field(j)
					data[columnName(field)] = fieldValue.Interface()
				}
				continue
			}
			data[columnName(field)] = fieldValue.Interface()
		}
		return data
	case reflect.Map:
		var data Args
		for _, key := range value.MapKeys() {
			data[key.String()] = value.MapIndex(key).Interface()
		}
		return data
	default:
		panic(fmt.Errorf("AsMap: unsupported type %s", t.Kind()))
	}
}

// AsMilliseconds convert duration to milliseconds
func AsMilliseconds(duration int64) float64 {
	return float64(duration) / float64(time.Millisecond)
}

// AsSeconds convert duration to seconds
func AsSeconds(duration int64) float64 {
	return float64(duration) / float64(time.Second)
}

// Ptr return pointer of v
func Ptr[T any](v T) *T {
	return &v
}

// PtrToValue return the value of pointer
func PtrToValue[T any](v *T) any {
	if v == nil {
		return nil
	}
	return *v
}
