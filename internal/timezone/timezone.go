package timezone

import "time"

// Vietnam timezone (UTC+7), loaded once at startup.
var VN *time.Location

func init() {
	VN = time.FixedZone("Asia/Ho_Chi_Minh", 7*60*60)
}

// Now returns the current time in Vietnam timezone (UTC+7).
func Now() time.Time {
	return time.Now().In(VN)
}
