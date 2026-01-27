package dagorinit

import (
	"hotel/utils"
	"strings"
	"time"

	dagor "hotel/dagor"
)

var log = utils.GetLogger("dagor-init")

var dagorParams = dagor.DagorParam{
	// Set the parameters accordingly
	NodeName: "",
	BusinessMap: map[string]int{
		"search-hotel":  1,
		"reserve-hotel": 2,
	},
	EntryService:                 false, // update this
	IsEnduser:                    false, // update this
	QueuingThresh:                2 * time.Millisecond,
	AdmissionLevelUpdateInterval: 10 * time.Millisecond,
	Alpha:                        0.05,
	Beta:                         0.01,
	Umax:                         15,
	Bmax:                         2,
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
	}

	node := dagor.NewDagorNode(dagorParams)

	return node
}
