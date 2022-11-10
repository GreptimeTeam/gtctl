package utils

import (
	"strings"
)

func SplitImageURL(imageURL string) (string, string) {
	// TODO(zyy17): validation?
	split := strings.Split(imageURL, ":")
	if len(split) != 2 {
		return "", ""
	}

	return split[0], split[1]
}
