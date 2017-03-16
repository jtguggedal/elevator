package backup

import (
	"encoding/json"
	"fmt"
	"os"
)

func SaveToFile(data interface{}) {
	file, err := os.Create("internalOrdersBackup.json")
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

func ReadFromFile(data interface{}) {
	file, err := os.Open("internalOrdersBackup.json")
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
