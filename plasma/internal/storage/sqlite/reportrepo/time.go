package reportrepo

import "time"

func parseRequiredTime(value string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, value)
}
