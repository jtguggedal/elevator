package file

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

func ReadFromFile() {

}
