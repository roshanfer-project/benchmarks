package dagorinit

import (
	"bufio"
	"os"
	"social/utils"
	"strconv"
	"strings"
	"time"

	dagor "social/dagor"
)

var log = utils.GetLogger("dagor-init")

var dagorParams = dagor.DagorParam{
	// Set the parameters accordingly
	NodeName: "",
	BusinessMap: map[string]int{
		"copose-post":        1,
		"read-user-timeline": 2,
		"read-home-timeline": 3,
	},
	EntryService:                 false, // update this
	IsEnduser:                    false, // update this
	QueuingThresh:                1824 * time.Microsecond,
	AdmissionLevelUpdateInterval: 13800 * time.Microsecond,
	Alpha:                        0.45,
	Beta:                         0.01,
	Umax:                         15,
	Bmax:                         3,
	Debug:                        false, // update this
	NumUsers:                     200,
	UseSyncMap:                   true,
	AddmissionUpdateN:            80,
}

func loadEnvFile(params dagor.DagorParam) dagor.DagorParam {
	file, err := os.Open("../env-setter.env")
	if err != nil {
		log.Info("env-setter file not found")
		return params
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if key == "Alpha" {
				if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
					params.Alpha = floatValue
					log.Info("[loadEnvFile]", "key", key, "value", params.Alpha)
				} else {
					panic(err)
				}
			} else if key == "Beta" {
				if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
					params.Beta = floatValue
					log.Info("[loadEnvFile]", "key", key, "value", params.Beta)
				} else {
					panic(err)
				}
			} else if key == "QueuingThresh" {
				if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
					intValue := int64(floatValue)
					params.QueuingThresh = time.Duration(intValue) * time.Microsecond
					log.Info("[loadEnvFile]", "key", key, "value", params.QueuingThresh)
				} else {
					panic(err)
				}
			} else if key == "AdmissionLevelUpdateInterval" {
				if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
					intValue := int64(floatValue)
					params.AdmissionLevelUpdateInterval = time.Duration(intValue) * time.Microsecond
					log.Info("[loadEnvFile]", "key", key, "value", params.AdmissionLevelUpdateInterval)
				} else {
					panic(err)
				}
			} else if key == "AddmissionUpdateN" {
				if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
					intValue := int(floatValue)
					params.AddmissionUpdateN = intValue
					log.Info("[loadEnvFile]", "key", key, "value", params.AddmissionUpdateN)
				} else {
					panic(err)
				}
			} else {
				panic("Unsupported environment variable type")
			}
		}

	}

	return params
}

func GetDagorNode(name string, entry, enduser bool) *dagor.Dagor {
	dagorParams.NodeName = name
	dagorParams.EntryService = entry
	dagorParams.IsEnduser = enduser

	if enduser {
		if utils.GetEnvVar("DAGOR_CLIENT_DEBUG", false) == "true" {
			log.Debug("[GetDagorNode] turning on dagor debug", "service", name)
			dagorParams.Debug = true
		}
	} else {
		if utils.GetEnvVar("DAGOR_"+strings.ToUpper(name)+"_DEBUG", false) == "true" {
			log.Debug("[GetDagorNode] turning on dagor debug", "service", name)
			dagorParams.Debug = true
		}
	}

	if utils.GetEnvVar("DAGOR_DEBUG", false) == "true" {
		log.Debug("[GetDagorNode] turning on dagor debug", "service", name)
		dagorParams.Debug = true
	}

	alpha := utils.GetEnvVar("Alpha", false)
	if alpha != "" {
		dagorParams.Alpha = utils.StrToFloat64(alpha)
		log.Info("Alpha got updated", "alpha", alpha)
	}
	beta := utils.GetEnvVar("Beta", false)
	if beta != "" {
		dagorParams.Beta = utils.StrToFloat64(beta)
		log.Info("Beta got updated", "beta", beta)
	}

	node := dagor.NewDagorNode(dagorParams)

	return node
}
