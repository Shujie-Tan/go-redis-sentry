package redis_sentry

func stringInSlice(key string, s []string) bool {

	for _, v := range s {
		if key == v {
			return true
		}
	}
	return false
}
