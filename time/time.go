package time

import (
	"time"
)

// Convert Time Format that compatible for insert time to database
func ConvertTimeFormat(reqTime time.Time) string {
	return reqTime.UTC().Format("2006-01-02 15:04:05")
}

// GetTimezoneInSeconds is get timezone from time
func GetTimezoneInSeconds(waktu time.Time) int {
	_, tz := waktu.Zone()
	// return waktu.f
	return tz
}

// ConvertTimezoneHour is check and convert timezone to hour if seconds
func ConvertTimezoneHour(tz int32) int32 {
	if tz > 1000 {
		tz /= 3600
	}
	return tz
}

// ResetAddTimezone is reset time to UTC and add timezone in seconds
func ResetAddTimezone(waktu time.Time, tz int32) time.Time {
	tz = ConvertTimezoneSeconds(tz)
	loc := time.FixedZone("id", int(tz))
	waktu = waktu.UTC()
	waktu = waktu.In(loc)
	return waktu
}

// ConvertTimezoneSeconds is check and convert timezone to seconds if hours
func ConvertTimezoneSeconds(tz int32) int32 {
	if tz < 1000 {
		tz *= 3600
	}
	return tz
}

// ConvertTimeToRFC3339 is change time to RFC3339 format
func ConvertTimeToRFC3339(waktu time.Time) string {
	return waktu.Format(time.RFC3339)
}

// ConvertRFC3339ToTime RFC3339 format to time
func ConvertRFC3339ToTime(waktu string) time.Time {
	var resp time.Time
	if waktu != "" {
		resp, _ = time.Parse(time.RFC3339, waktu)
	}
	return resp
}

func ConvertTimeMillisecond(reqTime time.Time) int64 {
	millisecond := int64(time.Nanosecond) * reqTime.UnixNano() / int64(time.Millisecond)
	return millisecond
}

func ConvertLocalTimeFormat(reqTime time.Time) string {
	return reqTime.Format("2006-01-02 15:04:05")
}
