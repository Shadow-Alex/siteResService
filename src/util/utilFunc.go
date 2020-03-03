/*
  Package util provide general utility functions
*/

package util

import (
	jsoniter "github.com/json-iterator/go"
	"io"
	"os"
	"fmt"
	"crypto/md5"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

// GetUUID return string of uuid
func GetUUID() string {
	var uuidSTR string
	uu, err := uuid.NewUUID()
	if err != nil {
		log.WithFields(log.Fields{
			"error info": err.Error(),
		}).Error("get uuid failed")
		uuidSTR = "failed-to-get-uuid"
	} else {
		uuidSTR = uu.String()
	}

	return uuidSTR
}

// ToJson for convert to json
func ToJson(data interface{}) string {
	json, err := jsoniter.Marshal(data)
	if err != nil {
		log.WithFields(log.Fields{
			"error info": err.Error(),
		}).Error("form to json failed")

		return ""
	}

	return string(json)
}

// GetMD5 return md5 string
func GetMD5(str string) string {
	md5Handle := md5.New()
	_, err := io.WriteString(md5Handle, str)
	if err != nil {
		log.WithFields(log.Fields{
			"error info": err.Error(),
		}).Error("get md5 failed")

		return ""
	}
	md5Str := fmt.Sprintf("%x", md5Handle.Sum(nil))

	return md5Str
}

// PathExists returns success flag and error info
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}

	return false, err
}