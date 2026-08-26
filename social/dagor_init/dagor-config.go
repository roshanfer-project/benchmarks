package dagorinit

import (
	"social/utils"
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
