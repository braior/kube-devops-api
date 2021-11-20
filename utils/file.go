package utils

import (
	"io/ioutil"
	"path"
)

// GetFileNameWithSuffix according to filename suffix,return a filename string slice and error
func GetFileNameWithSuffix(pathName, suffix string) ([]string, error) {
	var s []string
	rd, err := ioutil.ReadDir(pathName)
	if err != nil {
		return s, err
	}

	for _, fi := range rd {
		if fi.IsDir() {
			break

		}
		if path.Ext(fi.Name()) == suffix {
			s = append(s, fi.Name())
		}
	}
	return s, nil
}