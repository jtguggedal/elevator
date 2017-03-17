package backup

import (
	"encoding/json"
	"fmt"
	"os"
)

const (
	BackupInternal 	= 1
	BackupExternal	= 2
)

type BackupType int

func SaveToFile(data interface{}, t BackupType) {
	file, err := os.Create("backup.json")
	if t == BackupInternal {
		file, err = os.Create("internalOrdersBackup.json")
	} else {
		file, err = os.Create("externalOrdersBackup.json")
	}
	if err != nil {
		fmt.Println("Error when saving file:", err)
	}
	buf, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Error when converting to json:", err)
	}
	file.Write(buf)
	file.Close()
}

func ReadFromFile(data interface{}, t BackupType) {
	file, err := os.Create("backup.json")
	if t == BackupInternal {
		file, err = os.Open("internalOrdersBackup.json")
	} else {
		file, err = os.Open("externalOrdersBackup.json")
	}
	buf := make([]byte, 1024)
	n, err := file.Read(buf)
	if err != nil {
		fmt.Println("Error reading from file:", err)
	}
	err = json.Unmarshal(buf[:n], &data)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
	}
}
