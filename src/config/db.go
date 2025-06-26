package config

import (
	"fmt"
	"os"
)

func CreateDBFileIfNotExists() error {
	const fileName = "./falkie.db"

	info, err := os.Stat(fileName)
	if err == nil {
		if info.IsDir() {
			return fmt.Errorf("%s exists and is a directory", fileName)
		}
		return nil // file exists and is not a directory
	}
	if !os.IsNotExist(err) {
		return err // some other error
	}

	// Create the file
	f, err := os.Create(fileName)
	if err != nil {
		return err
	}
	return f.Close()
}
