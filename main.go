package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"syscall"

	"github.com/rdoorn/gohelper/influxdbhelper"
	"github.com/tarm/serial"
)

type Parser struct {
	influxdb      *influxdbhelper.Handler
	obisReference string
	data          P1Data
}

type P1Data struct {
	Timestamp                     string
	DeliveredToClientTariff1      float64
	DeliveredToClientTariff2      float64
	DeliveredByClientTariff1      float64
	DeliveredByClientTariff2      float64
	DeliveredToClientCurrent      float64
	DeliveredByClientCurrent      float64
	DeliveredToClientGas          float64
	DeliveredToClientGasTimestamp string
}

func main() {

	p1Path, ok := os.LookupEnv("P1_SERIAL_PATH")
	if !ok {
		p1Path = "/dev/ttyUSB0"
	}

	config := &serial.Config{
		Name:        p1Path,
		Baud:        9600,
		ReadTimeout: 20,
		Size:        7,
		Parity:      serial.ParityEven,
        StopBits:    1,
	}

	log.Printf("Connecting to %+v", config)
	stream, err := serial.OpenPort(config)
	if err != nil {
		log.Fatal(err)
	}
	reader := bufio.NewReader(stream)
	log.Println("setting up parser...")
	parser := &Parser{
		influxdb: influxdbhelper.New("telegraf"),
	}

	// loop till exit
	sigterm := make(chan os.Signal, 10)
	signal.Notify(sigterm, os.Interrupt, syscall.SIGTERM)

	for {
		line, _ := reader.ReadString('\n')
		parser.Parse(line)

		select {
		case <-sigterm:
			log.Printf("Program killed by signal!")
			return
		default:
		}
	}
}

func (p *Parser) Parse(s string) {
	if len(s) == 0 {
		return
	}
	log.Printf("parsing line: %s\n", s)
	log.Printf("parsing line: %+v\n", []byte(s))

	if id, hasID := getObisReference(s); hasID {
		p.obisReference = id
	}

	switch p.obisReference {
	case "1-0:1.8.1":
		log.Printf("tariff1 to: %s", s)
		p.data.DeliveredToClientTariff1 = GetValue(s)
	case "1-0:1.8.2":
		log.Printf("tariff2 to: %s", s)
		p.data.DeliveredToClientTariff2 = GetValue(s)
	case "1-0:2.8.1":
		log.Printf("tariff1 by: %s", s)
		p.data.DeliveredByClientTariff1 = GetValue(s)
	case "1-0:2.8.2":
		log.Printf("tariff2 by: %s", s)
		p.data.DeliveredByClientTariff2 = GetValue(s)
	case "1-0:1.7.0":
		log.Printf("curr to: %s", s)
		p.data.DeliveredToClientCurrent = GetValue(s)
	case "1-0:2.7.0":
		log.Printf("curr to: %s", s)
		p.data.DeliveredByClientCurrent = GetValue(s)
	case "0-1:24.3.0":
		log.Printf("gas by: %s", s)
		p.data.DeliveredToClientGas = GetValue(s)
	}

	if s[0] == '!' {
		log.Printf("Send Data: %+v", p.data)

		// sent power collected to nuts
		tags := map[string]string{
			"source": "dsmr",
			"metric": "kwh",
			"type":   "electricity",
		}
		fields := map[string]interface{}{
			"delivered_low":  p.data.DeliveredToClientTariff1,
			"delivered_high": p.data.DeliveredToClientTariff2,
			"returned_low":   p.data.DeliveredByClientTariff1,
			"returned_high":  p.data.DeliveredByClientTariff2,
		}
		log.Printf("sending fields: %+v\n", fields)
		err := p.influxdb.Insert("electricity", tags, fields)
		if err != nil {
			log.Printf(err.Error())
		}

		tags = map[string]string{
			"source": "dsmr",
			"metric": "m2",
			"type":   "gas",
		}

		fields = map[string]interface{}{
			"delivered": p.data.DeliveredToClientGas,
		}

		log.Printf("sending fields: %+v\n", fields)
		err = p.influxdb.Insert("gas", tags, fields)
		if err != nil {
			log.Printf(err.Error())
		}
	}

}

func getObisReference(s string) (string, bool) {
	// 0-0:96.1.1(ccc)
	r := regexp.MustCompile(`(\d+-\d+:\d+\.\d+\.\d+)\(`)
	m := r.FindStringSubmatch(s)
	if len(m) > 1 {
		return m[1], true
	}
	return "", false
}

func GetValue(s string) float64 {
	r := regexp.MustCompile(`.*\((\d+\.\d+)[\*\)]`)
	m := r.FindStringSubmatch(s)
	if len(m) != 2 {
		return 0
	}
	if f, err := strconv.ParseFloat(m[1], 64); err == nil {
		fmt.Printf("val: %f\n", f)
		return f
	}
	return 0
}

/*
{
  "id": 2864217,
  "timestamp": "2020-10-24T22:12:13.990Z",
  "electricity_delivered_1": "12972.117",
  "electricity_returned_1": "00746.379",
  "electricity_delivered_2": "14730.900",
  "electricity_returned_2": "01722.431",
  "electricity_currently_delivered": "0000.43",
  "electricity_currently_returned": "0000.00",
  "phase_currently_delivered_l1": null,
  "phase_currently_delivered_l2": null,
  "phase_currently_delivered_l3": null,
  "extra_device_timestamp": "2020-10-24T22:00:00Z",
  "extra_device_delivered": "08950.179",
  "phase_currently_returned_l1": null,
  "phase_currently_returned_l2": null,
  "phase_currently_returned_l3": null
}
*/

