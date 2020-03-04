/*
	package standalone  for running advService alone without micro service
*/

package standalone

import (
	"os"
	"os/exec"
	"sync"
	"time"
	"strconv"
	"strings"
	"bufio"
	"encoding/csv"

	"github.com/PuerkitoBio/goquery"
	"github.com/astaxie/beego/orm"
	log "github.com/sirupsen/logrus"

	mc "siteResService/src/mysqlClient"
	sc "siteResService/src/scheduler"
	tk "siteResService/src/task"
	cm "siteResService/src/common"
	ut "siteResService/src/util"
)


// StandAlone for stand alone running
type StandAlone struct {
	db 				*mc.MySQLClient
	task			*tk.ServiceTask
	siteResFile		*os.File  // store site resource data, not include spec and set
	siteSpecFile	*os.File  // store site specifications data
	siteSetFile		*os.File  // store site set meal data
	lastOffset      int64  // store last offset when last query
	lastStartTime	time.Time  // store last start time in query period
	lastTimeStamp	int64  // store last time in timestamp
}

var instance *StandAlone
var initStandAloneOnce sync.Once

// GetStandAloneInstance returns StandAlone instance pointer
func GetStandAloneInstance(db *mc.MySQLClient, task *tk.ServiceTask) *StandAlone {
	initStandAloneOnce.Do(func() {
		instance = new(StandAlone)
		instance.init(db, task)

		log.Info("init stand alone site resource service instance success...")
	})

	return instance
}

// init stand alone model
func (sa *StandAlone) init(db *mc.MySQLClient, task *tk.ServiceTask) {
	sa.db = db
	sa.lastOffset = 0
	sa.lastTimeStamp = 0
	sa.task = task

	sa.initSiteResultFile()  // init csv result file
}

// listPros for get landingURL
func (sa *StandAlone) listPros(maxID uint64, startTime int64, offset int64) *[]orm.Params {
	cond := orm.NewCondition()
	cond = cond.And("id__gt", maxID).Or("update_time__gt", startTime)
	ces := sa.db.CustomizedDBQueryMultiColumn("wc_cargo_ext", cond, 0)

	return ces
}

// addInfoToChan returns the num of item which added into channel
func (sa *StandAlone) addInfoToChan(items *[]orm.Params) int64 {
	if len(*items) <= 0 {
		return 0
	}

	for _, item := range *items {
		conns := orm.NewCondition()
		conns = conns.And("cargo_id", item["CargoId"])
		if sa.db.IsExist("wc_cargo_materials", conns) {
			log.Info("this cargoID has resources already, continue next")

			continue
		}

		landingURL := strings.TrimSpace(item["LandingUrl"].(string))
		sa.task.SubChan <- landingURL

		// only for debug
		log.WithFields(log.Fields{
			"landingURL":	item["LandingUrl"].(string),
		}).Debug("get landing url from db")
	}

	return int64(len(*items))
}

// TODO: need to reconstruct
// GetProsFromDB for get pros from of db
func (sa *StandAlone) GetProsFromDB() {
	var num int64  // query number
	maxID := sa.db.CustomizedDBQueryMax("wc_cargo_materials", orm.NewCondition(), "cargo_ext_id", 0)
	startTime := time.Now().Add(time.Hour * -4).Unix()
	offset := int64(0)  // init offset of query
	for {
		// set query condition
		items := sa.listPros(maxID, startTime, offset)
		num = sa.addInfoToChan(items)
		//num = sa.generateAddr(ids)
		// TODO： if query no more new data from db yesterday, then reset offset and startTime to run previous data
		if num == 0 {
			maxID = sa.db.CustomizedDBQueryMax("wc_cargo_materials", orm.NewCondition(), "cargo_ext_id", 0)
			startTime = time.Now().Unix()
			offset = 0

			continue
		}
		offset = offset + num

		// debug, for slow down http request frequency
		time.Sleep(time.Duration(cm.DBQueryGap) * time.Second)
	}
}

// initSiteFile for init site file to save process result
func initSiteFile(fileName string, titles []string) *os.File {
	fileInstance, err := os.OpenFile("data/" + fileName, os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		if os.IsNotExist(err) {
			fileInstance, err = os.Create("data/" + fileName)
			if err != nil {
				log.WithFields(log.Fields{
					"file":		"data/" + fileName,
					"error":	err.Error(),
				}).Panic("failed to create site file")
			}

			fileInstance.WriteString("\xEF\xBB\xBF")  // prevent Chinese garbled code
			w := csv.NewWriter(fileInstance)
			w.Write(titles)
			w.Flush()

			log.WithFields(log.Fields{
				"file":	"data/" + fileName,
			}).Info("site file is not exist, create success")
		} else {
			log.WithFields(log.Fields{
				"error":	err.Error(),
			}).Panic("failed to open site file")
		}
	}

	log.Info("init site file success...")

	return fileInstance
}

// InitSiteResultFile for init site Result file
func (sa *StandAlone) initSiteResultFile() {
	// init site resource file
	titles := []string{"pageURLMD5", "pageURL", "creatTime", "coverInJson", "title", "price", "descriptions", "template"}
	sa.siteResFile = initSiteFile(cm.SiteResourceFile, titles)
	// init site specifications file
	titles = []string{"pageURLMD5", "type", "mapping", "specifications"}
	sa.siteSpecFile = initSiteFile(cm.SiteSpecFile, titles)
	// init site set meals file
	titles = []string{"pageURLMD5", "setMeal"}
	sa.siteSetFile = initSiteFile(cm.SiteSetFile, titles)
}

// closeFileTGT for close target file
func (sa *StandAlone) CloseFileTGT() {
	if sa.siteResFile != nil {
		sa.siteResFile.Close()

		log.Info("siteResFile close success...")
	}
	if sa.siteSpecFile != nil {
		sa.siteSpecFile.Close()

		log.Info("siteSpecFile close success...")
	}
	if sa.siteSetFile != nil {
		sa.siteSetFile.Close()

		log.Info("siteSetFile close success...")
	}
}

// GetPageURLFromFile for get source page url from source file
func (sa *StandAlone) GetPageURLFromFile(fileSCR string) {
	file, err := os.Open("data/" + fileSCR)
	if err != nil {
		log.WithFields(log.Fields{
			"error":	err.Error(),
		}).Fatal("couldn't open the source file")

		return
	}

	r := bufio.NewScanner(file)
	for r.Scan() {
		// Read each record from csv file
		pageURL := r.Text()
		sa.task.SubChan <- pageURL

		// only for debug
		log.WithFields(log.Fields{
			"pageURL":	pageURL,
		}).Debug("get pageURL from file")
	}

	file.Close()

	log.Info("complete read all data from source file")
}

// chDirMod for change file's mod
func chDirMod(dir string) {
	log.WithFields(log.Fields{
		"dir":	dir,
	}).Info("all resources of this web page have been downloaded, chmod 755")
	exec.Command("chmod", "-R", "755", dir).Run()
}

// scriptTackle for tackle script description
func scriptTackle(desc string) string{
	if strings.Contains(desc, "script") {
		//pi.Desc = strings.ReplaceAll(pi.Desc, "script", "--")
		doc, _ := goquery.NewDocumentFromReader(strings.NewReader(desc))
		doc.Find("script").Remove()
		doc.Find("head").Remove()
		desc, _ = doc.Html()
		desc = strings.ReplaceAll(desc, "<html>", "<div>")
		desc = strings.ReplaceAll(desc, "</html>", "</div>")
	}

	return desc
}

// generateProInfoSlice generate slice from ProInfo struct
func generateProInfoSlice(proInfo *cm.ProInfo) []string {
	var cover string
	if len(proInfo.Cover) > 0 {
		cover = ut.ToJson(proInfo.Cover)
	}

	var proStr []string
	proStr = append(proStr, ut.GetMD5(proInfo.PageURL))
	proStr = append(proStr, proInfo.PageURL)
	proStr = append(proStr, strconv.FormatInt(time.Now().Unix(),10))
	proStr = append(proStr, cover)
	proStr = append(proStr, proInfo.Title)
	proStr = append(proStr, proInfo.Price)
	proStr = append(proStr, scriptTackle(proInfo.Desc))
	proStr = append(proStr, proInfo.Template)

	return proStr
}

// writeCSVFile for write data to csv file
func writeCSVFile(w *csv.Writer, data [][]string) {
	for _, d := range data {
		err := w.Write(d)
		if err != nil {
			log.WithFields(log.Fields{
				"error":	err.Error(),
			}).Error("write data to csv file failed")

			return
		}

		w.Flush()
	}
}

// TaskSaveResultToFile for save site resource to target file
func (sa *StandAlone) TaskSaveResultToFile(data *sc.DataBlock) {
	proInfo := data.Message.(*cm.ProInfo)

	w := csv.NewWriter(sa.siteResFile)
	writeCSVFile(w, [][]string{generateProInfoSlice(proInfo)})

	w = csv.NewWriter(sa.siteSpecFile)
	writeCSVFile(w, proInfo.Spec)

	w = csv.NewWriter(sa.siteSetFile)
	writeCSVFile(w, proInfo.Set)

	// chmod
	//fp := proInfo.Cover[0].URL
	//dir := strings.ReplaceAll(fp[:strings.LastIndex(fp, "/") + 1], cm.ImagePrefixUploads, cm.ImageDir)
	//chDirMod(dir)

	log.Info("finish csv file writing")
}

// ShowStandaloneADV for standalone file mode to debug
func (sa *StandAlone) ShowStandaloneADV() {
	for {
		select {
		case msg := <- sa.task.SubChan:
			log.WithFields(log.Fields{
				"pageURL":	msg,
			}).Info("handle result")
		}
	}
}