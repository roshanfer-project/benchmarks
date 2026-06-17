package dagorinit

import (
	"faninlb/utils"
	"strings"
	"time"

	dagor "faninlb/dagor"
)

var log = utils.GetLogger("dagor-init")

var dagorParams = dagor.DagorParam{
	NodeName:                     "",
	BusinessMap:                  map[string]int{
		"f1": 1,
		"g1": 2,
	},
	EntryService:                 false,
	IsEnduser:                    false,
	QueuingThresh:                8 * time.Millisecond,
	AdmissionLevelUpdateInterval: 10 * time.Millisecond,
	Alpha:                        0.45,
	Beta:                         0.01,
	Umax:                         15,
	Bmax:                         2,
	Debug:                        false,
	NumUsers:                     200,
	UseSyncMap:                   true,
	AddmissionUpdateN:            80,
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
