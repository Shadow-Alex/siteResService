package data

import (
	"encoding/csv"
	"github.com/astaxie/beego/logs"
	"net/http"
	"os"
)

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func ImportData(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		err := r.ParseForm()
		if err != nil {
			w.Write([]byte("failed: " + err.Error() + "\n"))
			return
		}

		dataPath := r.Form.Get("file")
		logs.Debug("data path: ", dataPath)

		if len(dataPath) == 0 {
			w.Write([]byte("failed: can't open file or no such file\n"))
			return
		}

		if !fileExists(dataPath) {
			w.Write([]byte("failed: can't open file or no such file\n"))
			return
		}

		if !parse(dataPath) {
			w.Write([]byte("failed: may not csv or format error\n"))
			return
		}

		w.Write([]byte("success\n"))
	}
}

func parse(file string) bool {
	csvfile, err := os.Open(file)
	if err != nil {
		logs.Debug("Couldn't open the csv file", err)
		return false
	}

	r := csv.NewReader(csvfile)
	records, err := r.ReadAll()
	if err != nil {
		logs.Debug("Can't read csv file", err)
	}

	logs.Debug(records)

	// TODO: 数据入库,update by item id

	return true
}
