package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"math"
	"net"
	"net/http"
	"time"
)

//Define a struct for you collector that contains pointers
//to prometheus descriptors for each metric you wish to expose.
//Note you can also include fields of other types if they provide utility
//but we just won't be exposing them as metrics.
type smartWattHourCollector struct {
	electricityMetric *prometheus.Desc
	voltageMetric     *prometheus.Desc
	wattMetric        *prometheus.Desc
}

var eportAddress string
var eportPort string

//You must create a constructor for you collector that
//initializes every descriptor and returns a pointer to the collector
func newSmartWattHourMeterCollector() *smartWattHourCollector {
	return &smartWattHourCollector{
		electricityMetric: prometheus.NewDesc("electricityMetric",
			"Shows electricityMetric",
			nil, nil,
		),
		voltageMetric: prometheus.NewDesc("voltageMetric",
			"Shows voltageMetric",
			nil, nil,
		),
		wattMetric: prometheus.NewDesc("wattHourMetric",
			"Shows wattHourMetric",
			nil, nil,
		),
	}
}

//Each and every collector must implement the Describe function.
//It essentially writes all descriptors to the prometheus desc channel.
func (collector *smartWattHourCollector) Describe(ch chan<- *prometheus.Desc) {

	//Update this section with the each metric you create for a given collector
	ch <- collector.electricityMetric
	ch <- collector.voltageMetric
	ch <- collector.wattMetric
}

//Collect implements required collect function for all promehteus collectors
func (collector *smartWattHourCollector) Collect(ch chan<- prometheus.Metric) {

	//Implement logic here to determine proper metric value to return to prometheus
	//for each descriptor or call other functions that do so.

	conn, err := net.Dial("tcp", eportAddress+":"+eportPort)
	if err != nil {
		fmt.Printf("Fail to connect, %s\n", err)
		return
	}
	currentMetric, voltageMetric, wattMetric := getSmartWattHourMetersInfo(conn)

	//Write latest value for each metric in the prometheus metric channel.
	//Note that you can pass CounterValue, GaugeValue, or UntypedValue types here.
	ch <- prometheus.MustNewConstMetric(collector.electricityMetric, prometheus.CounterValue, currentMetric)
	ch <- prometheus.MustNewConstMetric(collector.voltageMetric, prometheus.CounterValue, voltageMetric)
	ch <- prometheus.MustNewConstMetric(collector.wattMetric, prometheus.CounterValue, wattMetric)

}

func getSmartWattHourMetersInfo(c net.Conn) (float64, float64, float64) {
	defer c.Close()

	send_string := []byte{0x03, 0x04, 0x00, 0x00, 0x00, 0x32, 0x70, 0x3d}

	c.Write(send_string)

	buf := make([]byte, 100)
	_, err := c.Read(buf)

	if err != nil {
		fmt.Printf("Fail to read data, %s\n", err)
	}
	//fmt.Printf("%0X\n",buf)
	//fmt.Printf("%0X\n", buf[3:7])
	//fmt.Printf("%0X\n", buf[19:23])
	//fmt.Printf("%0X\n", buf[39:43])

	//data := []byte{66, 106, 179, 86}
	voltageMeterBits := binary.BigEndian.Uint32(buf[3:7])
	voltageMeter := math.Float32frombits(voltageMeterBits)
	//fmt.Println(voltageMeter)

	currentMetricBits := binary.BigEndian.Uint32(buf[19:23])
	currentMetric := math.Float32frombits(currentMetricBits)
	//fmt.Println(currentMetric)

	wattMetricBits := binary.BigEndian.Uint32(buf[39:43])
	wattMetric := math.Float32frombits(wattMetricBits)
	//fmt.Println(wattMetric)

	//fmt.Println(currentMetric, voltageMeter, wattHourMetric)
	fmt.Printf("%s %s:%g %s:%g %s:%g\n", time.Now().Format("2006-01-02 15:04:05"), "currentMetric", currentMetric, "voltageMeter", voltageMeter, "wattMetric", wattMetric)

	return float64(currentMetric), float64(voltageMeter), float64(wattMetric)
}

func main() {
	ip := flag.String("host", "100.127.255.136", "eport ip address")
	port := flag.String("port", "8899", "eport's port")
	listenPort := flag.String("listenPort", "8899", "exporter listen port")

	flag.Parse()

	eportAddress = *ip
	eportPort = *port

	//Create a new instance of the foocollector and
	//register it with the prometheus client.
	smartWattHourMeter := newSmartWattHourMeterCollector()
	prometheus.MustRegister(smartWattHourMeter)
	//
	////This section will start the HTTP server and expose
	////any metrics on the /metrics endpoint.
	http.Handle("/metrics", promhttp.Handler())
	fmt.Println("Beginning to serve on port: " + *listenPort)
	http.ListenAndServe(":"+*listenPort, nil)
	fmt.Println("end")
}
