package breakwaterinit

import (
	"bufio"
	"os"
	"social/utils"
	"strconv"
	"strings"

	bw "social/breakwater"
)

var bwConfigServer = bw.BWParameters{
	Verbose:          false,
	SLO:              12500,
	ClientExpiration: 10000,
	InitialCredits:   400,
	LoadShedding:     true,
	ServerSide:       true,
	AFactor:          1.64,
	BFactor:          4.4,
	RTT_MICROSECOND:  615,
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
	bwConfigClient.SLO = 12500
	bwConfigClient.ClientExpiration = 10000
	bwConfigClient.InitialCredits = 400
	bwConfigClient.LoadShedding = false
	bwConfigClient.ServerSide = false
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
				if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
					accepted.SLO = intValue
				}
			} else if key == "breakwaterClientExpiration" {
				if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
					accepted.ClientExpiration = intValue
				}
			} else if key == "breakwaterAFactor" {
				if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
					accepted.AFactor = floatValue
				}
			} else if key == "breakwaterBFactor" {
				if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
					accepted.BFactor = floatValue
				}
			} else if key == "breakwaterRTT_MICROSECOND" {
				if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
					accepted.RTT_MICROSECOND = intValue
				}
			} else {
				log.Info("[loadEnvFile] unsupported environment variable type", "key", key, "value", value)
			}
			log.Info("[loadEnvFile]", "key", key, "value", accepted)
		}
	}

	return accepted
}

func GetBreakwater(name string, enduser bool) *bw.Breakwater {
	var bwConfig *bw.BWParameters
	if !enduser {
		bwConfig = &bwConfigServer
		if utils.GetEnvVar("BREAKWATER_"+strings.ToUpper(name)+"_DEBUG", false) == "true" {
			log.Info("[GetPriceTable] turning on breakwater debug", "service", name)
			bwConfig.Verbose = true
		}
	} else {
		bwConfig = &bwConfigClient
		if utils.GetEnvVar("BREAKWATER_CLIENT_DEBUG", false) == "true" {
			bwConfig.Verbose = true
			log.Info("[GetPriceTable] turning on breakwater debug for end user", "service", name)
		}
	}

	if utils.GetEnvVar("BREAKWATER_DEBUG", false) == "true" {
		bwConfig.Verbose = true
		log.Info("[GetPriceTable] turning on breakwater debug for all users", "service", name)
	}

	bwConfig = loadEnvFile(bwConfig)

	return bw.InitBreakwater(*bwConfig)
}
