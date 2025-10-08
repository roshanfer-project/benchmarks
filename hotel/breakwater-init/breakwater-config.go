package breakwaterinit

import (
	"bufio"
	"hotel/utils"
	"os"
	"strconv"
	"strings"

	bw "hotel/breakwater"
)

const breakwaterSLO = 8000
const breakwaterClientExpiration = 300
const breakwaterA = 0.000
const breakwaterB = 0.2
const breakwaterRTT = 5000

const breakwaterInitialCredit = 100

var bwConfigServer = bw.BWParameters{
	Verbose:          false,
	SLO:              breakwaterSLO,
	ClientExpiration: breakwaterClientExpiration,
	InitialCredits:   breakwaterInitialCredit,
	LoadShedding:     true,
	ServerSide:       true,
	AFactor:          breakwaterA,
	BFactor:          breakwaterB,
	RTT_MICROSECOND:  breakwaterRTT,
	TrackCredits:     false,
}

/* var bwConfigClient = bw.BWParameters{
	Verbose:          false,
	SLO:              12500,
	ClientExpiration: 10000,
	InitialCredits:   400,
	LoadShedding:     false,
	ServerSide:       false,
	AFactor:          breakwaterA,
	BFactor:          breakwaterB,
	RTT_MICROSECOND:  breakwaterRTT.Microseconds(),
	TrackCredits:     breakwaterTrackCredit,
} */

var bwConfigClient = bw.BWParametersDefault

func init() {
	bwConfigClient.SLO = breakwaterSLO
	bwConfigClient.ClientExpiration = breakwaterClientExpiration
	bwConfigClient.InitialCredits = breakwaterInitialCredit
	bwConfigClient.LoadShedding = false
	bwConfigClient.ServerSide = false
	bwConfigClient.UseClientQueueLength = true
	bwConfigClient.UseClientTimeExpiration = false
	log.Info("Breakwater client configuration initialized")
}

var log = utils.GetLogger("breakwater-init")

func loadEnvFile(accepted *bw.BWParameters) *bw.BWParameters {
	file, err := os.Open("../env-setter.env")
	if err != nil {
		log.Info("env-setter file not found")
		return accepted
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if key == "breakwaterSLO" {
				if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
					accepted.SLO = int64(floatValue)
				} else {
					panic(err)
				}
			} else if key == "breakwaterClientExpiration" {
				if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
					accepted.ClientExpiration = int64(floatValue)
				} else {
					panic(err)
				}
			} else if key == "breakwaterAFactor" {
				if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
					accepted.AFactor = floatValue
				} else {
					panic(err)
				}
			} else if key == "breakwaterBFactor" {
				if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
					accepted.BFactor = floatValue
				} else {
					panic(err)
				}
			} else if key == "breakwaterRTT_MICROSECOND" {
				if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
					accepted.RTT_MICROSECOND = int64(floatValue)
				} else {
					panic(err)
				}
			} else {
				log.Info("[loadEnvFile] unsupported environment variable type", "key", key, "value", value)
				panic("Unsupported environment variable type: " + key)
			}
		}
	}

	return accepted
}

func GetBreakwater(name string, enduser bool) *bw.Breakwater {
	var bwConfig *bw.BWParameters
	if !enduser {
		bwConfig = &bwConfigServer
		if utils.GetEnvVar("BREAKWATER_"+strings.ToUpper(name)+"_DEBUG", false) == "true" ||
			utils.GetEnvVar("BREAKWATERD_"+strings.ToUpper(name)+"_DEBUG", false) == "true" {
			log.Info("[GetPriceTable] turning on breakwater debug", "service", name)
			bwConfig.Verbose = true
		}
	} else {
		bwConfig = &bwConfigClient
		if utils.GetEnvVar("BREAKWATER_CLIENT_DEBUG", false) == "true" ||
			utils.GetEnvVar("BREAKWATERD_CLIENT_DEBUG", false) == "true" {
			bwConfig.Verbose = true
			log.Info("[GetPriceTable] turning on breakwater debug for end user", "service", name)
		}
	}

	if utils.GetEnvVar("BREAKWATER_DEBUG", false) == "true" ||
		utils.GetEnvVar("BREAKWATERD_DEBUG", false) == "true" {
		bwConfig.Verbose = true
		log.Info("[GetPriceTable] turning on breakwater debug for all users", "service", name)
	}

	bwConfig = loadEnvFile(bwConfig)
	// log important parameters
	log.Info("[GetPriceTable] Breakwater configuration", "config", bwConfig)

	return bw.InitBreakwater(*bwConfig)
}
