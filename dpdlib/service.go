package dpdlib

import (
	"fmt"
	"github.com/fiorix/wsdl2go/soap"
	"log"
	"strings"
	"time"
)

type commonPoint struct {
	code      string
	latitude  string
	longitude string
	limits    normalizedLimits
	schedule  string
	cityCode  uint32
	state     string
}

type normalizedLimits struct {
	maxWeight    float32
	maxWidth     float32
	maxHeight    float32
	maxLength    float32
	dimensionSum float32
}

func NewDpdSdk(clientNumber int64, clientKey, countryCode string) DpdSdk {
	return &dpdSdk{
		clientNumber: clientNumber,
		clientKey:    clientKey,
		countryCode:  countryCode,
	}
}

type dpdSdk struct {
	clientNumber int64
	clientKey    string
	countryCode  string
}

type DpdSdk interface {
	GetOffers() []*ServiceCost
	GetPoints() []*commonPoint
	//getLimitedPoints(geography2 DPDGeography2)
	//getUnlimitedPoints(geography2 DPDGeography2)

}

func (dpdSdk) GetOffers() []*ServiceCost {
	client := soap.Client{
		URL:       "http://wstest.dpd.ru:80/services/calculator",
		Namespace: CalculatorNamespace,
	}

	service := NewDPDCalculator(&client)

	getServiceCost2Req := &GetServiceCost2{
		//TODO: implement req body
	}

	result, err := service.GetServiceCost2(getServiceCost2Req)

	if err != nil {
		log.Fatalf("%s", err.Error())
	}

	return result.Return
}

func (sdk dpdSdk) GetPoints() []*commonPoint {
	client := soap.Client{
		URL:       "http://wstest.dpd.ru/services/geography2",
		Namespace: GeographyNamespace,
	}

	service := NewDPDGeography2(&client)

	limitedPoints := sdk.getLimitedPoints(service)
	unlimitedPoints := sdk.getUnlimitedPoints(service)
	result := append(limitedPoints, unlimitedPoints...)

	return result
}

func (sdk dpdSdk) getLimitedPoints(service DPDGeography2) []*commonPoint {

	limitedPointsReq := &GetParcelShops{
		Ns: GeographyNamespace,
		Request: &DpdParcelShopRequest{
			Auth: &Auth{
				ClientNumber: &sdk.clientNumber,
				ClientKey:    &sdk.clientKey,
			},
			CountryCode: &sdk.countryCode,
		},
	}

	start := time.Now()

	result, err := service.GetParcelShops(limitedPointsReq)

	PrintTime("Dpd parcel shops req", start)

	if err != nil {
		log.Fatalf("%s", err.Error())
	}

	start = time.Now()

	commonPoints := convertDpdLimitedPoints2Common(result.Return.ParcelShop)

	PrintTime("Convert parcel shops", start)

	return commonPoints
}

func (sdk dpdSdk) getUnlimitedPoints(service DPDGeography2) []*commonPoint {
	unlimitedPointsReq := &GetTerminalsSelfDelivery2{
		Ns: GeographyNamespace,
		Auth: &Auth{
			ClientNumber: &sdk.clientNumber,
			ClientKey:    &sdk.clientKey,
		},
	}

	start := time.Now()

	result, err := service.GetTerminalsSelfDelivery2(unlimitedPointsReq)

	PrintTime("Dpd terminals req", start)

	if err != nil {
		log.Fatalf("%s", err.Error())
	}

	start = time.Now()

	commonPoints := convertDpdUnlimitedPoints2Common(result.Return.Terminal)

	PrintTime("Convert terminals", start)

	return commonPoints
}

func convertDpdUnlimitedPoints2Common(terminals []*TerminalSelf) []*commonPoint {
	var commonPoints []*commonPoint

	for _, terminal := range terminals {
		commonPoints = append(commonPoints, &commonPoint{
			code:      *terminal.TerminalCode,
			latitude:  fmt.Sprintf("%g", *terminal.GeoCoordinates.Latitude),
			longitude: fmt.Sprintf("%g", *terminal.GeoCoordinates.Longitude),
			schedule:  getStrSchedule(terminal.Schedule),
			cityCode:  uint32(*terminal.Address.CityId),
			state:     "Open",
		})
	}

	return commonPoints
}

func convertDpdLimitedPoints2Common(shops []*ParcelShop) []*commonPoint {
	var commonPoints []*commonPoint

	for _, shop := range shops {
		commonPoints = append(commonPoints, &commonPoint{
			code:      *shop.Code,
			latitude:  fmt.Sprintf("%g", *shop.GeoCoordinates.Latitude),
			longitude: fmt.Sprintf("%g", *shop.GeoCoordinates.Longitude),
			limits:    getValidLimits(shop.Limits),
			schedule:  getStrSchedule(shop.Schedule),
			cityCode:  uint32(*shop.Address.CityId),
			state:     *shop.State,
		})
	}

	return commonPoints
}

func getValidLimits(limits *Limits) normalizedLimits {
	var normalizedLimits normalizedLimits

	if limits == nil {
		return normalizedLimits
	}

	if limits.MaxWeight != nil {
		normalizedLimits.maxWeight = float32(*limits.MaxWeight)
	}

	if limits.MaxWidth != nil {
		normalizedLimits.maxWidth = float32(*limits.MaxWidth)
	}

	if limits.MaxLength != nil {
		normalizedLimits.maxLength = float32(*limits.MaxLength)
	}

	if limits.MaxHeight != nil {
		normalizedLimits.maxHeight = float32(*limits.MaxHeight)
	}

	if limits.DimensionSum != nil {
		normalizedLimits.dimensionSum = float32(*limits.DimensionSum)
	}

	return normalizedLimits
}

func getStrSchedule(schedule []*Schedule) string {

	if schedule == nil || len(schedule) == 0 {
		return ""
	}

	//Обычно SelfDelivery - самый последний элемент массива
	lastItem := schedule[len(schedule)-1:]

	if len(lastItem) > 0 && strings.Compare(*lastItem[0].Operation, "SelfDelivery") == 0 {
		var tt []string
		for _, schedule := range lastItem[0].Timetable {
			tt = append(tt, *schedule.WeekDays+" "+*schedule.WorkTime)
		}

		return strings.Join(tt, " ")
	}

	return getStrSchedule(schedule[:len(schedule)-1])
}

func PrintTime(description string, start time.Time) {
	end := time.Now()
	fmt.Printf("Action: %s, Duration: %f\n", description, end.Sub(start).Seconds())
}
