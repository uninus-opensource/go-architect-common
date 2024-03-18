package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/uninus-opensource/uninus-go-architect-common/flags"
	"github.com/uninus-opensource/uninus-go-architect-common/config/configetcd"
	"github.com/uninus-opensource/uninus-go-architect-common/config/configzk"
	"github.com/uninus-opensource/uninus-go-architect-common/microservice"
)

type StdConfig struct {
	ServiceName string
	ServiceRoot string
	ConfigHosts []string
	ConfigData  map[string]string
	EventPath   string
	eventHook   func()
}

const (
	//in app name --storage name
	Certificates = "certificates"
	CAcert       = "CAcert"
	TLScrt       = "TLScrt"
	TLSkey       = "TLSkey"
	JWTkey       = "JWTkey"

	Database = "database"
	DBhost   = "dbhost"
	DBport   = "dbport"
	DBname   = "dbname"
	DBuid    = "dbuid"
	DBpwd    = "dbpwd"
	DBdriver = "dbdriver"

	RDhost     = "rdhost"
	RDport     = "rdport"
	RDpassword = "rdpassword"

	GProjectId = "gprojectid"
	GCredFile  = "gcredfile"

	SubsNum = "subsnum"

	Addr_listen = "addr_listen"
	Addr_debug  = "addr_debug"
)

var ValidKeys = map[string]bool{
	Database:     true,
	DBname:       true,
	DBhost:       true,
	DBport:       true,
	DBuid:        true,
	DBpwd:        true,
	DBdriver:     true,
	Certificates: true,
	CAcert:       true,
	TLScrt:       true,
	TLSkey:       true,
	JWTkey:       true,
}

// global variable
var AppConfig StdConfig
var AppEvent = func() {}
var localConfig = false
var configFile = "service.conf"

func Get(key string, defval string) string {

	//find the key in the current defval map
	//
	v, ok := AppConfig.ConfigData[key]
	if !ok {
		if defval != "" {
			AppConfig.ConfigData[key] = defval
			v = defval
		}
	}
	return v
}

func GetA(key string, defval string) []string {

	//find the key in the current defval map
	//
	v := Get(key, defval)

	if len(v) == 0 {
		return nil
	}

	sep := ","

	if len(defval) == 1 {
		sep = defval
	}
	//return the string as []string
	//
	return strings.Split(v, sep)
}

func GetI(key string, defval int) int {

	//find the key in the current defval map
	//
	v := Get(key, strconv.Itoa(defval))

	if res, err := strconv.Atoi(v); err == nil {
		return res
	} else {
		return -1
	}
}

func GetB(key string, defval bool) bool {

	//find the key in the current defval map
	//
	v := Get(key, strconv.FormatBool(defval))

	if res, err := strconv.ParseBool(v); err == nil {
		return res
	} else {
		return false
	}
}

// test the newly created map for data validity
func KeyTest(key string) bool {
	_, ok := ValidKeys[key]

	return ok
}

func (sc *StdConfig) ConfigPath() string {
	return fmt.Sprintf("%s/%s", sc.ServiceRoot, sc.ServiceName)
}

func (sc *StdConfig) SetChangeNotificationFunc(f func()) {
	sc.eventHook = f
}

// loads a standard config file,
// located in the same path as the app,
// containing only servicename and confighosts array
func (sc *StdConfig) open() error {
	configFile, err := os.Open(configFile)
	if err != nil {
		return err
	}
	defer configFile.Close()

	jsonParser := json.NewDecoder(configFile)

	err = jsonParser.Decode(&sc)

	return err
}

func (sc *StdConfig) LoadConfigFile(file string) bool {
	if file != "" {
		configFile = file
	}
	return sc.LoadConfig()
}

func (sc *StdConfig) LoadConfigLocal() bool {
	localConfig = true
	return sc.LoadConfig()
}

func (sc *StdConfig) LoadConfig() bool {

	//1.
	//We need to get:
	//   -servicename
	//   -confighosts
	//from somewhere. look in ENV vars, if not found, then look in FILE
	//
	s, ok := os.LookupEnv(strings.ToUpper("bbg_servicename"))
	if ok {
		sc.ServiceName = s
	} else {
		err := sc.open()
		if err != nil {
			log.Printf("error reading service.conf file. %+v\n", err)
			return false
		}
		//if still not found then exit:
		if sc.ServiceName == "" {
			return false
		}
	}

	s, ok = os.LookupEnv(strings.ToUpper("bbg_confighosts"))
	if ok {
		sc.ConfigHosts = strings.Split(s, ",")
	} else {
		sc.open()

		//if still not found then exit:
		// if len(sc.ConfigHosts) == 0 {
		// 	return false
		// }
	}

	if !localConfig && len(sc.ConfigHosts) > 0 {
		// 2.
		//- connect to a etcd node instance.
		//- locate and monitor a service node for notification event
		//- receive svcNode's data in the form of map[string]map
		hosts := sc.ConfigHosts
		svcNode := sc.ConfigPath()

		var err error

		var CfgData map[string]string
		switch microservice.GetOsEnv(flags.UNINUS_DISCOVERY_ENV_NAME) {
		case flags.UNINUS_DISCOVERY_MODE_ZK:
			CfgData, err = configzk.ZKConnectAndListen(hosts, svcNode, sc.onZKChangeEvent)
			if err != nil {
				log.Printf("zk error %v\n", err)
				return false
			}

			if sc.ConfigData == nil {
				sc.ConfigData = make(configzk.ConfigFormat)
			}

			for k, v := range CfgData {
				sc.ConfigData[k] = v
			}

		default:
			CfgData, err = configetcd.ETCDConnectAndListen(hosts, svcNode, sc.onETCDChangeEvent)
			if err != nil {
				log.Printf("etcd  error %v\n", err)
				return false
			}

			if sc.ConfigData == nil {
				sc.ConfigData = make(configetcd.ConfigFormat)
			}

			for k, v := range CfgData {
				sc.ConfigData[k] = v
			}
		}

	}

	//Scan the ENV vars matching our prefix,
	//add these new data to the existing map:
	//
	for _, e := range os.Environ() {
		pair := strings.Split(e, "=")
		key := strings.ToLower(pair[0])

		if strings.HasPrefix(key, "bbg_") {
			key = strings.Trim(key, "bbg_")

			sc.ConfigData[key] = pair[1]
		}
	}

	//IMPORTANT:
	//returned data is a map with string Key and string value
	//log.Println("StdConfig", sc.ConfigData)

	return true
}

func (sc *StdConfig) onZKChangeEvent(nodename string, dataMap configzk.ConfigFormat) {
	sc.ConfigData = dataMap
	sc.EventPath = nodename
	if sc.eventHook != nil {
		sc.eventHook()
	}
}

func (sc *StdConfig) onETCDChangeEvent(nodename string, dataMap configetcd.ConfigFormat) {
	sc.ConfigData = dataMap
	sc.EventPath = nodename
	if sc.eventHook != nil {
		sc.eventHook()
	}
}
