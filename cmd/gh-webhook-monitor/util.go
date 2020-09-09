package main

// mapSubexpNames maps regex capturing group names to corresponding matches
func mapSubexpNames(names, matches []string) map[string]string {
	//names, matches = names[1:], matches[1:]
	nameMatchMap := make(map[string]string, len(matches))
	for index := range names {
		nameMatchMap[names[index]] = matches[index]
	}
	return nameMatchMap
}
