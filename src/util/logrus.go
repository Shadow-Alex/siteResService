/*
  Package util provide general utility functions, this use github.com/sirupsen/logrus as log  for output format log.
*/

package util

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/astaxie/beego"
	rotatelogs "github.com/lestrrat/go-file-rotatelogs"
	"github.com/rifflock/lfshook"
	"github.com/sirupsen/logrus"

	cm "siteResService/src/common"
)

// InitLogrus init and config logrus and return, it gets parameter from conf/app.conf about:
// log director, log file name, log max age, log level to config logrus,
// if success, a *lfshook.LfsHook can be add to log Hook.
func InitLogrus() *lfshook.LfsHook {
	// make log dir
	logDir := beego.AppConfig.DefaultString("log::dir", cm.LogDir)
	if errDir := os.MkdirAll(logDir, os.ModePerm); errDir != nil {
		fmt.Printf("create log dir faild ! log dir: %v, error: %v\n", logDir, errDir.Error())

		return nil
	}

	// log path
	logFile := beego.AppConfig.DefaultString("log::fileName", cm.LogFilename)
	baseLogPath := path.Join(logDir, logFile)
	logMaxAge := beego.AppConfig.DefaultInt64("log::maxAge", cm.LogWithMaxAge)
	writer, errR := rotatelogs.New(
		baseLogPath+".%Y-%m-%d",                                      		// log form
		rotatelogs.WithClock(rotatelogs.Local),								// set time zone CST
		rotatelogs.WithLinkName(baseLogPath),                         		// soft link to log path
		rotatelogs.WithMaxAge(time.Duration(logMaxAge * 24) * time.Hour), 	// log file maximum keep time
		//rotatelogs.WithRotationTime(time.Duration(24) * time.Hour),     	// use default: 24 * time.Hour
	)
	if errR != nil {
		fmt.Printf("config rotate log local file system failed !  error: %v\n", errR.Error())

		return nil
	}

	// set log output log level, log only output higher level log then it sets
	// log output level:  Trace=6; Debug=5; Info=4; Warn=3; Error=2; Fatal=1; Panic=0
	logLevel := beego.AppConfig.DefaultInt("log::level", cm.LogLevel)
	logrus.SetLevel(logrus.Level(logLevel))

	// set stdout to /dev/null (redirect), do not output to stdout
	nullSrc, errN := os.OpenFile(os.DevNull, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if errN == nil {
		writerNull := bufio.NewWriter(nullSrc)
		logrus.SetOutput(writerNull)
	}

	fmt.Printf("init logrus success... \nlog path: %v, log maxage: %v, log level: %v\n", baseLogPath, logMaxAge, logLevel)

	// set each level log output destination to writer
	return lfshook.NewHook(lfshook.WriterMap{
		logrus.TraceLevel: writer,
		logrus.DebugLevel: writer,
		logrus.InfoLevel:  writer,
		logrus.WarnLevel:  writer,
		logrus.ErrorLevel: writer,
		logrus.FatalLevel: writer,
		logrus.PanicLevel: writer,
	}, &logrus.TextFormatter{TimestampFormat: "2006-01-02 15:04:05.000"})
}
